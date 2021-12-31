// Copyright (c) 2014 The SurgeMQ Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"errors"
	"fmt"
	"github.com/rfyiamcool/go-timewheel"
	"io"
	"xmqtt/auth"
	"xmqtt/utils/cnt"
	"xmqtt/utils/recover"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"xmqtt/consts"
	"xmqtt/sessions"
	"xmqtt/utils/log"
	"xmqtt/utils/surgemq/message"
)

var (
	ErrInvalidConnectionType  error = errors.New("service: Invalid connection type")
	ErrInvalidSubscriber      error = errors.New("service: Invalid subscriber")
	ErrInvalidClientId        error = errors.New("service：invalid ClientId")
	ErrInvalidUsername        error = errors.New("service：invalid Username")
	ErrInvalidPassword        error = errors.New("service：invalid Password")
	ErrBufferNotReady         error = errors.New("service: buffer is not ready")
	ErrBufferInsufficientData error = errors.New("service: buffer has insufficient data")
)

const (
	DefaultKeepAlive      = 300
	DefaultConnectTimeout = 2
	DefaultAckTimeout     = 20
	DefaultTimeoutRetries = 3
)
// Server is a library implementation of the MQTT server that, as best it can, complies
// with the MQTT 3.1 and 3.1.1 specs.
type Server struct {
	// The number of seconds to keep the connection live if there's no data.
	// If not set then default to 5 mins.
	KeepAlive int

	// The number of seconds to wait for the CONNECT message before disconnecting.
	// If not set then default to 2 seconds.
	ConnectTimeout int

	// The number of seconds to wait for any ACK messages before failing.
	// If not set then default to 20 seconds.
	AckTimeout int

	// The number of times to retry sending a packet if ACK is not received.
	// If no set then default to 3 retries.
	TimeoutRetries int

	MaxBufferSize int

	// SessionsProvider is the session store that keeps all the Session objects.
	// This is the store to check if CleanSession is set to 0 in the CONNECT message.
	// If not set then default to "mem".
	SessionsProvider string

	// sessMgr is the sessions manager for keeping track of the sessions
	sessMgr *sessions.Manager

	// The quit channel for the server. If the server detects that this channel
	// is closed, then it's a signal for it to shutdown as well.
	//quit chan struct{}

	// The tlsquit channel for the tls server. If the tls server detects that this channel
	// is closed, then it's a signal for it to shutdown as well.
	//tlsquit chan struct{}

	// The tlssquit channel for the tls server. If the tls server detects that this channel
	// is closed, then it's a signal for it to shutdown as well.
	//tlssquit chan struct{}

	ln     net.Listener
	lnTls  net.Listener

	nodeID string

	// A list of services created by the server. We keep track of them so we can
	// gracefully shut them down if they are still alive when the server goes down.
	// clientID --> service
	svcMap sync.Map

	// Mutex for updating svcMap
	mu sync.Mutex

	// A indicator on whether this server is running
	running int32

	// A indicator on whether this TLS server is running
	tlsrunning int32

	// A indicator on whether this secure TLS server is running
	tlssrunning int32

	// A indicator on whether this server has already checked configuration
	configOnce sync.Once

	subs []interface{}
	qoss []byte

	PublicNetworkIP string
	tw *timewheel.TimeWheel
	pm PMPool

	hostIP string
	mqttAddr string
	tlsMqttAddr string
	sTlsMqttAddr string
}

func (this *Server) setNodeID(nodeID string) {
	this.nodeID = nodeID
}

func (this *Server) Run(uri string) error {
	go this.listenAndServe(uri)
	return nil
}

// ListenAndServe listents to connections on the URI requested, and handles any
// incoming MQTT client sessions. It should not return until Close() is called
// or if there's some critical error that stops the server from running. The URI
// supplied should be of the form "protocol://host:port" that can be parsed by
// url.Parse(). For example, an URI could be "tcp://0.0.0.0:1883".
func (this *Server) listenAndServe(uri string) error {
	defer atomic.CompareAndSwapInt32(&this.running, 1, 0)

	if !atomic.CompareAndSwapInt32(&this.running, 0, 1) {
		return fmt.Errorf("server.ListenAndServe: Server is already running")
	}
	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("server.ListenAndServe | url.Parse failed:%s", err.Error())
		return err
	}

	this.ln, err = net.Listen(u.Scheme, u.Host)
	if err != nil {
		return err
	}
	defer this.ln.Close()

	log.Debugf("server.ListenAndServe | server is ready...")

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		conn, err := this.ln.Accept()

		if err != nil {

			// Borrowed from go1.3.3/src/pkg/net/http/server.go:1699
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Errorf("server.ListenAndServe | Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}

		go this.handleConnection(conn)
	}

	return err
}

// Close terminates the server by shutting down all the client connections and closing
// the listener. It will, as best it can, clean up after itself.
func (this *Server) Close() error {
	// We then close the net.Listener, which will force Accept() to return if it's
	// blocked waiting for new connections.
	if this.ln != nil {
		this.ln.Close()
	}

	if this.lnTls != nil {
		this.lnTls.Close()
	}

	this.svcMap.Range(func(key, value interface{}) bool {
		svc := value.(*service)
		svc.reason = consts.LogReasonValServerClose
		val := this.deleteMapAndLogout(svc)
		if val != nil{
			log.Errorf("deleteMapAndLogout | event: %s, reason: %s, err: %s", cnt.DeviceLogout, consts.LogReasonValServerClose, val.Error())
		}

		this.sessMgr.Del(svc.sess.ID())
		svc.newStopConnect()
		svc = nil
		return true
	})

	if this.sessMgr != nil {
		this.sessMgr.Close()
	}

	return nil
}

// HandleConnection is for the broker to handle an incoming connection from a client
func (this *Server) handleConnection(c io.Closer) (svc *service, err error) {
	defer recover.PanicHandler()

	if c == nil {
		return nil, ErrInvalidConnectionType
	}

	err = this.checkConfiguration()
	if err != nil {
		log.Errorf("checkConfiguration failed, error %s", err.Error())
		return nil, err
	}

	conn, ok := c.(net.Conn)
	if !ok {
		log.Errorf("get conn error %s", ErrInvalidConnectionType.Error())
		return nil, ErrInvalidConnectionType
	}

	defer func() {
		if err != nil {
			conn.Close()
			c.Close()
		}
	}()

	// To establish a connection, we must
	// 1. Read and decode the message.ConnectMessage from the wire
	// 2. If no decoding errors, then authenticate using username and password.
	//    Otherwise, write out to the wire message.ConnackMessage with
	//    appropriate error.
	// 3. If authentication is successful, then either create a new session or
	//    retrieve existing session
	// 4. Write out to the wire a successful message.ConnackMessage message

	// Read the CONNECT message from the wire, if error, then check to see if it's
	// a CONNACK error. If it's CONNACK error, send the proper CONNACK error back
	// to client. Exit regardless of error type.

	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(this.ConnectTimeout)))
	resp := message.NewConnackMessage()
	req, err := getConnectMessage(conn)

	if err != nil {
		log.Debugf("server.handleConnection | getConnectMessage err:", err.Error())
		if cerr, ok := err.(message.ConnackCode); ok {
			resp.SetReturnCode(cerr)
			resp.SetSessionPresent(false)
			if err = writeMessage(conn, resp); err != nil {
				return nil, err
			}
			return nil, err
		}
		return nil, err
	}

	if req.KeepAlive() == 0 {
		req.SetKeepAlive(5 * 60)
	}

	userName := string(req.Username())
	clientID := string(req.ClientId())
	passWord := string(req.Password())

	var productKey string
	var deviceName string

	temp := strings.Split(userName, "&")
	if len(temp) == 2 {
		productKey = temp[1]
		deviceName = temp[0]
	} else {
		resp.SetReturnCode(message.ErrBadClientIdOrUsernameOrPassword)
		resp.SetSessionPresent(false)
		if err = writeMessage(conn, resp); err != nil {
			return nil, err
		}
		return nil, ErrInvalidUsername
	}

	//// clientId检查  clientID+"|securemode=3,algorithm=hmac-sha256,timeStamp=1533176065|"
	//if strings.Count(clientID, "|") != 2 || strings.Count(clientID, "=") != 3 {
	//	resp.SetReturnCode(message.ErrBadClientIdOrUsernameOrPassword)
	//	resp.SetSessionPresent(false)
	//	if err = writeMessage(conn, resp); err != nil {
	//		return nil, err
	//	}
	//	return nil, ErrInvalidClientId
	//}

	if passWord == "" {
		resp.SetReturnCode(message.ErrBadClientIdOrUsernameOrPassword)
		resp.SetSessionPresent(false)
		if err = writeMessage(conn, resp); err != nil {
			return nil, err
		}
		return nil, ErrInvalidPassword
	}

	splitClientId := strings.Split(clientID, "|")[0]
	svc = &service{
		id:         atomic.AddUint64(&gsvcid, 1),
		client:     false,
		clientAddr: conn.RemoteAddr().String(),
		productKey: productKey,
		deviceName: deviceName,
		nodeID:     this.nodeID,

		keepAlive:      int(req.KeepAlive()),
		keepAliveCH:    make(chan int32),
		connectTimeout: this.ConnectTimeout,
		ackTimeout:     this.AckTimeout,
		timeoutRetries: this.TimeoutRetries,

		conn:     conn,
		done:     make(chan struct{}),
		csMgr:    &connectivityStateManager{},
		clientId: splitClientId,
		tw: this.tw,
	}


	svc.pm.init()
	log.Debugf("server.handleConnection | productKey %s deviceName %s clientAddr %s ClientId %s Username %s Password %s", svc.productKey, svc.deviceName, svc.clientAddr, clientID, userName, passWord)

	key := strings.Join([]string{svc.productKey, ":", svc.deviceName}, "")
	svcOld, ok := this.svcMap.Load(key)
	if ok {
		log.Debugf("server.handleConnection | Tick off old %s:%s connection", svc.productKey, svc.deviceName)
		svcOldService := svcOld.(*service)
		svcOldService.reason = consts.LogReasonValTickOffDevice
		val := this.deleteMapAndLogout(svcOldService)
		if val != nil{
			log.Errorf("deleteMapAndLogout | event: %s, reason: %s, err: %s", cnt.DeviceLogout, consts.LogReasonValTickOffDevice, val.Error())
		}
		this.sessMgr.Del(svcOldService.sess.ID())
		svcOldService.newStopConnect()
		svcOldService = nil
	}

	rst, err := auth.Login(auth.LoginInput{
		ClientId: clientID,
		Password: passWord,
	})

	log.Debugf("server.handleConnection | productKey %s deviceName %s deviceLogin rst:%+v err:%v", svc.productKey, svc.deviceName, rst, err)
	svc.loginValue = rst.Value
	if err != nil  {
		resp.SetReturnCode(message.ErrAuthServerLoginFailed)
		resp.SetSessionPresent(false)
		if err = writeMessage(conn, resp); err != nil {
			return nil, err
		}
		return nil, err
	}

	err = this.getSession(svc, req, resp)
	if err != nil {
		log.Errorf("server.handleConnection | productKey %s deviceName % getSession err:%v", svc.productKey, svc.deviceName, err)
		return nil, err
	}

	resp.SetReturnCode(message.ConnectionAccepted)
	if err = writeMessage(c, resp); err != nil {
		log.Errorf("productKey %s deviceName %s writeMessage failed: %s", svc.productKey, svc.deviceName, err.Error())
		return nil, err
	}

	svc.inStat.increment(int64(req.Len()))
	svc.outStat.increment(int64(resp.Len()))

	if err := svc.start(this); err != nil {
		log.Debugf("server.handleConnection | productKey %s deviceName %s start service error", svc.productKey, svc.deviceName)
		this.sessMgr.Del(svc.sess.ID())
		svc.newStopConnect()
		svc = nil
		log.Errorf("server.handleConnection | productKey %s deviceName %s start service, error: %s", svc.productKey, svc.deviceName, err.Error())
		return nil, err
	}


	this.storeMap(key, svc)
	log.Debugf("server.handleConnection | productKey %s deviceName %s Connection established.", svc.productKey, svc.deviceName)
	return svc, nil
}

func (this *Server) storeMap(key string, val *service) {
	this.svcMap.Store(key, val)
}

func (this *Server) deleteMapAndLogout(svc *service) error {
	key := strings.Join([]string{svc.productKey, svc.deviceName}, ":")
	this.svcMap.Delete(key)

	err := auth.Logout(auth.LogoutInput{
		DeviceName: svc.deviceName,
		ProductKey: svc.productKey,
		Value: svc.loginValue,
	})
	if err != nil{
		log.Errorf("deleteMapAndLogout failed, err: %s, DeviceName: %s, ProductKey: %s, loginValue: %s", err.Error(), svc.deviceName, svc.productKey, svc.loginValue)
	}
	return err
}

func (this *Server) checkConfiguration() error {
	var err error

	this.configOnce.Do(func() {
		if this.KeepAlive == 0 {
			this.KeepAlive = DefaultKeepAlive
		}

		if this.ConnectTimeout == 0 {
			this.ConnectTimeout = DefaultConnectTimeout
		}

		if this.AckTimeout == 0 {
			this.AckTimeout = DefaultAckTimeout
		}

		if this.TimeoutRetries == 0 {
			this.TimeoutRetries = DefaultTimeoutRetries
		}

		if this.SessionsProvider == "" {
			this.SessionsProvider = "mem"
		}

		this.sessMgr, err = sessions.NewManager(this.SessionsProvider)
		if err != nil {
			return
		}
		return
	})

	return err
}

func (this *Server) getSession(svc *service, req *message.ConnectMessage, resp *message.ConnackMessage) error {
	// If CleanSession is set to 0, the server MUST resume communications with the
	// client based on state from the current session, as identified by the client
	// identifier. If there is no session associated with the client identifier the
	// server must create a new session.
	//
	// If CleanSession is set to 1, the client and server must discard any previous
	// session and start a new one. This session lasts as long as the network c
	// onnection. State data associated with this session must not be reused in any
	// subsequent session.

	var err error

	// Check to see if the client supplied an ID, if not, generate one and set
	// clean session.
	if len(req.ClientId()) == 0 {
		req.SetClientId([]byte(fmt.Sprintf("internalclient%d", svc.id)))
		req.SetCleanSession(true)
	}

	cid := string(req.ClientId())

	// If CleanSession, or no existing session found, then create a new one
	if svc.sess == nil {
		if svc.sess, err = this.sessMgr.New(cid); err != nil {
			return err
		}

		resp.SetSessionPresent(false)

		if err := svc.sess.Init(req); err != nil {
			return err
		}
	}

	return nil
}
