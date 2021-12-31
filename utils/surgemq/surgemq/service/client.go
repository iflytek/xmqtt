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
	"context"
	"fmt"
	"net"
	"net/url"
	"sync/atomic"
	"time"

	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"git.iflytek.com/HY_XIoT/core/utils/log"
	"git.iflytek.com/HY_XIoT/core/xfyun/surgemq/message"
	"git.iflytek.com/HY_XIoT/core/xfyun/surgemq/surgemq/sessions"
	"git.iflytek.com/HY_XIoT/core/xfyun/surgemq/surgemq/topics"
)

const (
	minKeepAlive = 30
)

// Client is a library implementation of the MQTT client that, as best it can, complies
// with the MQTT 3.1 and 3.1.1 specs.
type Client struct {
	// The number of seconds to keep the connection live if there's no data.
	// If not set then default to 5 mins.
	KeepAlive int

	// The number of seconds to wait for the CONNACK message before disconnecting.
	// If not set then default to 2 seconds.
	ConnectTimeout int

	// The number of seconds to wait for any ACK messages before failing.
	// If not set then default to 20 seconds.
	AckTimeout int

	// The number of times to retry sending a packet if ACK is not received.
	// If no set then default to 3 retries.
	TimeoutRetries int

	svc *service

	mu sync.Mutex
}

// Custom connection using specified client IP and port
// func DialCustom(network, address string, timeout time.Duration, localIP []byte, localPort int) (net.Conn, error) {
func (this *Client) DialCustom(network, address string, port int, timeout time.Duration, localIP []byte) (net.Conn, error) {
	//netAddr := &net.TCPAddr{Port: localPort}
	// use default port
	//netAddr := &net.TCPAddr{Port: 9001}
	netAddr := &net.TCPAddr{Port: port}

	if len(localIP) != 0 {
		netAddr.IP = localIP
	}

	//fmt.Println("client.DialCustom | netAddr:", netAddr)
	log.Debugf("client.DialCustom | netAddr:%s", netAddr)

	d := net.Dialer{Timeout: timeout, LocalAddr: netAddr}
	return d.Dial(network, address)
}

// Decimal to hexadecimal
func (this *Client) DecHex(n int64) string {
	if n < 0 {
		log.Errorln("Decimal to hexadecimal error: the argument must be greater than zero.")
		return ""
	}
	if n == 0 {
		return "0"
	}
	hex := map[int64]int64{10: 65, 11: 66, 12: 67, 13: 68, 14: 69, 15: 70}
	s := ""
	for q := n; q > 0; q = q / 16 {
		m := q % 16
		if m > 9 && m < 16 {
			m = hex[m]
			s = fmt.Sprintf("%v%v", string(m), s)
			continue
		}
		s = fmt.Sprintf("%v%v", m, s)
	}
	return s
}

// BytesCombine 多个[]byte数组合并成一个[]byte
func (this *Client) BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

// Int to Byte
func (this *Client) IntToByte(num int64) []byte {
	var buffer bytes.Buffer
	err := binary.Write(&buffer, binary.BigEndian, num)
	if err != nil {
		log.Errorf("client.IntToByte | bytes.Buffer Write err:%v", err)
		os.Exit(1)
	}
	return buffer.Bytes()
}

// Connect is for MQTT clients to open a connection to a remote server. It needs to
// know the URI, e.g., "tcp://127.0.0.1:1883", so it knows where to connect to. It also
// needs to be supplied with the MQTT CONNECT message.
// If the virtual IP is used for pressure measurement,
// You need to connect to the local IP.
func (this *Client) Connect(uri string, msg *message.ConnectMessage) (err error) {
	log.Debugf("After inter <Connect> | uri:%s", uri)
	this.checkConfiguration()

	if msg == nil {
		return fmt.Errorf("msg is nil")
	}

	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("client | Connect url.Parse err:%s", err.Error())
		return err
	}

	if u.Scheme != "tcp" {
		return ErrInvalidConnectionType
	}

	conn, err := net.Dial(u.Scheme, u.Host)
	if err != nil {
		log.Debugf("client | Connect Scheme:%s Host:%s", u.Scheme, u.Host)
		log.Errorf("client | Connect net.Dial err:%s", err.Error())
		return err
	}

	defer func() {
		if err != nil {
			conn.Close()
		}
	}()

	if msg.KeepAlive() < minKeepAlive {
		msg.SetKeepAlive(minKeepAlive)
	}

	log.Debugf("client.Connect | Connect ClientId:%s", string(msg.ClientId()))
	log.Debugf("client.Connect | Connect Username:%s", string(msg.Username()))
	log.Debugf("client.Connect | Connect Password:%s", string(msg.Password()))

	if err = writeMessage(conn, msg); err != nil {
		log.Errorf("client.Connect | Connet writeMessage err:%s", err.Error())
		return err
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(this.ConnectTimeout)))

	resp, err := getConnackMessage(conn)
	if err != nil {
		log.Errorf("client.Connect | Connect getConnackMessage err:%s", err.Error())
		return err
	}

	log.Debugf("client.Connect | Connect getConnackMessage resp:%s", resp.String())

	if resp.ReturnCode() != message.ConnectionAccepted {
		//fmt.Println("resp.ReturnCode() != message.ConnectionAccepted err")
		return resp.ReturnCode()
	}

	var productKey string
	var deviceName string

	//username检查
	temp := strings.Split(string(msg.Username()), "&")
	if len(temp) == 2 {
		productKey = temp[1]
		deviceName = temp[0]
	}

	this.svc = &service{
		id:         atomic.AddUint64(&gsvcid, 1),
		client:     true,
		productKey: productKey,
		deviceName: deviceName,
		conn:       conn,
		done:       make(chan struct{}),
		csMgr:      &connectivityStateManager{},

		keepAlive:      int(msg.KeepAlive()),
		connectTimeout: this.ConnectTimeout,
		ackTimeout:     this.AckTimeout,
		timeoutRetries: this.TimeoutRetries,
	}

	err = this.getSession(this.svc, msg, resp)
	if err != nil {
		log.Errorf("client.Connect | Connect this.getSession err:%s", err.Error())
		return err
	}

	p := topics.NewMemProvider()
	log.Debugf("client.Connect | register ID:%s", string(this.svc.sess.ID()))
	topics.Register(this.svc.sess.ID(), p)

	this.svc.topicsMgr, err = topics.NewManager(this.svc.sess.ID())
	if err != nil {
		log.Errorf("client.Connect | Connect topics.NewManager err:%s", err.Error())
		return err
	}

	this.svc.csMgr.updateState(Ready)
	if err = this.svc.start(); err != nil {
		log.Errorf("client.Connect | Connect this.svc.start err:%s", err.Error())
		this.svc.stop()
		return err
	}

	this.svc.inStat.increment(int64(msg.Len()))
	this.svc.outStat.increment(int64(resp.Len()))

	return nil
}

func (this *Client) ConnectTLS(uri string, msg *message.ConnectMessage) (err error) {
	fmt.Println("Inter <ConnectTLS>")
	log.Debugf("After inter <ConnectTLS> | uri:%s", uri)
	this.checkConfiguration()

	if msg == nil {
		return fmt.Errorf("msg is nil")
	}

	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("client.ConnectTLS | ConnectTLS url.Parse err:%s", err.Error())
		return err
	}

	config := &tls.Config{
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
	}

	if u.Scheme != "tcp" {
		return ErrInvalidConnectionType
	}

	conn, err := tls.Dial(u.Scheme, u.Host, config)
	if err != nil {
		log.Debugf("client.ConnectTLS | ConnectTLS Scheme:%s Host:%s", u.Scheme, u.Host)
		log.Errorf("client.ConnectTLS | ConnectTLS net.DialTCP err:%s", err.Error())
		return err
	}

	if msg.KeepAlive() < minKeepAlive {
		msg.SetKeepAlive(minKeepAlive)
	}

	log.Debugf("client.ConnectTLS | ConnectTLS ClientId:%s", string(msg.ClientId()))
	log.Debugf("client.ConnectTLS | ConnectTLS Username:%s", string(msg.Username()))
	log.Debugf("client.ConnectTLS | ConnectTLS Password:%s", string(msg.Password()))

	if err = writeMessage(conn, msg); err != nil {
		log.Errorf("client.ConnectTLS | ConnectTLS writeMessage err:%s", err.Error())
		return err
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(this.ConnectTimeout)))

	resp, err := getConnackMessage(conn)
	if err != nil {
		log.Errorf("client.ConnectTLS | ConnectTLS getConnackMessage err:%s", err.Error())
		return err
	}

	log.Debugf("client.ConnectTLS | ConnectTLS getConnackMessage resp:%s", resp.String())

	if resp.ReturnCode() != message.ConnectionAccepted {
		//fmt.Println("resp.ReturnCode() != message.ConnectionAccepted err")
		return resp.ReturnCode()
	}

	this.svc = &service{
		id:     atomic.AddUint64(&gsvcid, 1),
		client: true,
		conn:   conn,
		done:   make(chan struct{}),
		csMgr:  &connectivityStateManager{},

		keepAlive:      int(msg.KeepAlive()),
		connectTimeout: this.ConnectTimeout,
		ackTimeout:     this.AckTimeout,
		timeoutRetries: this.TimeoutRetries,
	}

	err = this.getSession(this.svc, msg, resp)
	if err != nil {
		log.Errorf("client.ConnectTLS | ConnectTLS this.getSession err:%s", err.Error())
		return err
	}

	p := topics.NewMemProvider()
	log.Debugf("client.ConnectTLS | register ID:%s", string(this.svc.sess.ID()))
	topics.Register(this.svc.sess.ID(), p)

	this.svc.topicsMgr, err = topics.NewManager(this.svc.sess.ID())
	if err != nil {
		log.Errorf("client.ConnectTLS | ConnectTLS topics.NewManager err:%s", err.Error())
		return err
	}

	this.svc.csMgr.updateState(Ready)
	if err = this.svc.start(); err != nil {
		log.Errorf("client.ConnectTLS | ConnectTLS this.svc.start err:%s", err.Error())
		this.svc.stop()
		return err
	}

	this.svc.inStat.increment(int64(msg.Len()))
	this.svc.outStat.increment(int64(resp.Len()))

	return nil
}

func (this *Client) ConnectSTLS(uri, rootCsr, clientCert, clientKey string, msg *message.ConnectMessage) (err error) {
	fmt.Println("Inter <ConnectSTLS>")
	log.Debugf("After inter <ConnectSTLS> | uri:%s", uri)
	this.checkConfiguration()

	if msg == nil {
		return fmt.Errorf("msg is nil")
	}

	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS url.Parse err:%s", err.Error())
		return err
	}

	certpool := x509.NewCertPool()
	perCerts, err := ioutil.ReadFile(rootCsr) // "cert/myopenssl.csr"
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS ioutil.ReadFile[%s] err:%s", rootCsr, err.Error())
	} else {
		certpool.AppendCertsFromPEM(perCerts)
		fmt.Println("0. read pemCerts Succes")
		log.Infoln("0. read pemCerts Succes")
	}

	cert, err := tls.LoadX509KeyPair(clientCert, clientKey) // "cert/client.crt"、"cert/client.key"
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS tls.LoadX509KeyPair[%s:%s] err:%s", clientCert, clientKey, err.Error())
		return err
	}
	fmt.Println("1. read client cert Success")
	log.Infoln("client.ConnectSTLS | 1. read client cert Success")

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS x509.ParseCertificate err:%s", err.Error())
	}
	fmt.Println("2. read cert.Leaf Success")
	log.Infoln("client.ConnectSTLS | 2. read cert.Leaf Success")

	config := &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: true,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}

	if u.Scheme != "tcp" {
		return ErrInvalidConnectionType
	}
	raddr, err := net.ResolveTCPAddr(u.Scheme, u.Host)
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS net.ResolveTCPAddr err:%s", err.Error())
		return err
	}

	conn, err := net.DialTCP(u.Scheme, nil, raddr)
	if err != nil {
		log.Debugf("client.ConnectSTLS | ConnectSTLS Scheme:%s Host:%s", u.Scheme, u.Host)
		log.Errorf("client.ConnectSTLS | ConnectSTLS net.DialTCP err:%s", err.Error())
		return err
	}

	tlsConn := tls.Client(conn, config)

	defer func() {
		if err != nil {
			tlsConn.Close()
		}
	}()

	if msg.KeepAlive() < minKeepAlive {
		msg.SetKeepAlive(minKeepAlive)
	}

	log.Debugf("client.ConnectSTLS | ConnectSTLS ClientId:%s", string(msg.ClientId()))
	log.Debugf("client.ConnectSTLS | ConnectSTLS Username:%s", string(msg.Username()))
	log.Debugf("client.ConnectSTLS | ConnectSTLS Password:%s", string(msg.Password()))

	if err = writeMessage(tlsConn, msg); err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS writeMessage err:%s", err.Error())
		return err
	}

	tlsConn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(this.ConnectTimeout)))

	resp, err := getConnackMessage(tlsConn)
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS getConnackMessage err:%s", err.Error())
		return err
	}

	log.Debugf("client.ConnectSTLS | ConnectSTLS getConnackMessage resp:%s", resp.String())

	if resp.ReturnCode() != message.ConnectionAccepted {
		//fmt.Println("resp.ReturnCode() != message.ConnectionAccepted err")
		return resp.ReturnCode()
	}

	this.svc = &service{
		id:     atomic.AddUint64(&gsvcid, 1),
		client: true,
		conn:   tlsConn,
		done:   make(chan struct{}),
		csMgr:  &connectivityStateManager{},

		keepAlive:      int(msg.KeepAlive()),
		connectTimeout: this.ConnectTimeout,
		ackTimeout:     this.AckTimeout,
		timeoutRetries: this.TimeoutRetries,
	}

	err = this.getSession(this.svc, msg, resp)
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS this.getSession err:%s", err.Error())
		return err
	}

	p := topics.NewMemProvider()
	log.Debugf("client.ConnectSTLS | register ID:%s", string(this.svc.sess.ID()))
	topics.Register(this.svc.sess.ID(), p)

	this.svc.topicsMgr, err = topics.NewManager(this.svc.sess.ID())
	if err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS topics.NewManager err:%s", err.Error())
		return err
	}

	this.svc.csMgr.updateState(Ready)
	if err = this.svc.start(); err != nil {
		log.Errorf("client.ConnectSTLS | ConnectSTLS this.svc.start err:%s", err.Error())
		this.svc.stop()
		return err
	}

	this.svc.inStat.increment(int64(msg.Len()))
	this.svc.outStat.increment(int64(resp.Len()))

	return nil
}

func (this *Client) VirtualIPConnectOld(uri, VirtualIPPredix string, i int, port int, msg *message.ConnectMessage) (err error) {
	//fmt.Println("Inter <Connect>")
	log.Debugf("After intere <Connect> | uri:%s", uri)
	this.checkConfiguration()

	if msg == nil {
		return fmt.Errorf("msg is nil")
	}
	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("client | Connect url.Parse err:%s", err.Error())
		return err
	}

	if u.Scheme != "tcp" {
		return ErrInvalidConnectionType
	}

	VirtualIPPredixSplit := strings.Split(VirtualIPPredix, ".")
	//fmt.Println("VirtualIPPredixSplit:", VirtualIPPredixSplit)

	var VirtualIPByteArr []byte

	VirtualIPPredixSplitFirstSyllableInt, _ := strconv.Atoi(VirtualIPPredixSplit[0])
	//fmt.Println("VirtualIPPredixSplitFirstSyllableInt:", VirtualIPPredixSplitFirstSyllableInt)
	VirtualIPPredixSplitSecondSyllableInt, _ := strconv.Atoi(VirtualIPPredixSplit[1])
	//fmt.Println("VirtualIPPredixSplitSecondSyllableInt:", VirtualIPPredixSplitSecondSyllableInt)
	VirtualIPPredixSplitThirdSyllableInt, _ := strconv.Atoi(VirtualIPPredixSplit[2])
	//fmt.Println("VirtualIPPredixSplitThirdSyllableInt:", VirtualIPPredixSplitThirdSyllableInt)
	VirtualIPByteArr = append(append(append(append(VirtualIPByteArr, uint8(VirtualIPPredixSplitFirstSyllableInt)), uint8(VirtualIPPredixSplitSecondSyllableInt)), uint8(VirtualIPPredixSplitThirdSyllableInt)), uint8(i+1))

	conn, err := this.DialCustom(u.Scheme, u.Host, port, time.Second*10, VirtualIPByteArr)
	if err != nil {
		log.Debugf("client.VirtualIPConnect | Connect Scheme:%s Host:%s VirtualIPByteArr:%v", u.Scheme, u.Host, VirtualIPByteArr)
		log.Errorf("client.VirtualIPConnect | Connect DialCustom err:%s", err.Error())
		return err
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
		//conn.Close()
		log.Debugf("client.VirtualIPConnect | do not close this connect,keep this connection alived fo press test")
	}()

	if msg.KeepAlive() < minKeepAlive {
		msg.SetKeepAlive(minKeepAlive)
	}

	log.Debugf("client.VirtualIPConnect | Connect ClientId:%s", string(msg.ClientId()))
	log.Debugf("client.VirtualIPConnect | Connect Username:%s", string(msg.Username()))
	log.Debugf("client.VirtualIPConnect | Connect Password:%s", string(msg.Password()))

	if err = writeMessage(conn, msg); err != nil {
		log.Errorf("client.VirtualIPConnect | Connet writeMessage err:%s", err.Error())
		return err
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(this.ConnectTimeout)))

	resp, err := getConnackMessage(conn)
	if err != nil {
		log.Errorf("client.VirtualIPConnect | Connect getConnackMessage err:%s", err.Error())
		return err
	}

	log.Debugf("client.VirtualIPConnect | Connect getConnackMessage resp:%s", resp.String())

	if resp.ReturnCode() != message.ConnectionAccepted {
		//fmt.Println("resp.ReturnCode() != message.ConnectionAccepted err")
		return resp.ReturnCode()
	}

	this.svc = &service{
		id:     atomic.AddUint64(&gsvcid, 1),
		client: true,
		conn:   conn,
		done:   make(chan struct{}),
		csMgr:  &connectivityStateManager{},

		keepAlive:      int(msg.KeepAlive()),
		connectTimeout: this.ConnectTimeout,
		ackTimeout:     this.AckTimeout,
		timeoutRetries: this.TimeoutRetries,
	}
	//this.mu.Unlock()

	err = this.getSession(this.svc, msg, resp)
	if err != nil {
		log.Errorf("client.VirtualIPConnect | Connect this.getSession err:%s", err.Error())
		return err
	}

	p := topics.NewMemProvider()
	log.Debugf("client.VirtualIPConnect | register ID:%s", string(this.svc.sess.ID()))
	topics.Register(this.svc.sess.ID(), p)

	this.svc.topicsMgr, err = topics.NewManager(this.svc.sess.ID())
	if err != nil {
		log.Errorf("client | Connect topics.NewManager err:%s", err.Error())
		return err
	}

	this.svc.csMgr.updateState(Ready)
	if err = this.svc.start(); err != nil {
		log.Errorf("client | Connect this.svc.start err:%s", err.Error())
		this.svc.stop()
		return err
	}

	this.svc.inStat.increment(int64(msg.Len()))
	this.svc.outStat.increment(int64(resp.Len()))

	return nil
}

func (this *Client) VirtualIPConnect(uri string, VirtualIPByteArr []byte, port int, msg *message.ConnectMessage) (err error) {
	//fmt.Println("Inter <Connect>")
	log.Debugf("After intere <Connect> | uri:%s", uri)
	this.checkConfiguration()

	if msg == nil {
		return fmt.Errorf("msg is nil")
	}
	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("client | Connect url.Parse err:%s", err.Error())
		return err
	}

	if u.Scheme != "tcp" {
		return ErrInvalidConnectionType
	}
	log.Debugf("client.VirtualIPConnect | Connect Scheme:%s Host:%s VirtualIPByteArr:%v VirtualIPPort:%d", u.Scheme, u.Host, VirtualIPByteArr, port)
	conn, err := this.DialCustom(u.Scheme, u.Host, port, time.Second*10, VirtualIPByteArr)
	if err != nil {
		log.Debugf("client.VirtualIPConnect | Connect Scheme:%s Host:%s VirtualIPByteArr:%v VirtualIPPort:%d", u.Scheme, u.Host, VirtualIPByteArr, port)
		log.Errorf("client.VirtualIPConnect | Connect DialCustom err:%s", err.Error())
		return err
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
		//conn.Close()
		log.Debugf("client.VirtualIPConnect | do not close this connect,keep this connection alived fo press test")
	}()

	if msg.KeepAlive() < minKeepAlive {
		msg.SetKeepAlive(minKeepAlive)
	}

	log.Debugf("client.VirtualIPConnect | Connect ClientId:%s", string(msg.ClientId()))
	log.Debugf("client.VirtualIPConnect | Connect Username:%s", string(msg.Username()))
	log.Debugf("client.VirtualIPConnect | Connect Password:%s", string(msg.Password()))

	if err = writeMessage(conn, msg); err != nil {
		log.Errorf("client.VirtualIPConnect | Connet writeMessage err:%s", err.Error())
		return err
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(this.ConnectTimeout)))

	resp, err := getConnackMessage(conn)
	if err != nil {
		log.Errorf("client.VirtualIPConnect | Connect getConnackMessage err:%s", err.Error())
		return err
	}

	log.Debugf("client.VirtualIPConnect | Connect getConnackMessage resp:%s", resp.String())

	if resp.ReturnCode() != message.ConnectionAccepted {
		//fmt.Println("resp.ReturnCode() != message.ConnectionAccepted err")
		return resp.ReturnCode()
	}

	var productKey string
	var deviceName string

	//username检查
	temp := strings.Split(string(msg.Username()), "&")
	if len(temp) == 2 {
		productKey = temp[1]
		deviceName = temp[0]
	}
	this.svc = &service{
		id:         atomic.AddUint64(&gsvcid, 1),
		client:     true,
		productKey: productKey,
		deviceName: deviceName,
		conn:       conn,
		done:       make(chan struct{}),
		csMgr:      &connectivityStateManager{},

		keepAlive:      int(msg.KeepAlive()),
		connectTimeout: this.ConnectTimeout,
		ackTimeout:     this.AckTimeout,
		timeoutRetries: this.TimeoutRetries,
	}
	//this.mu.Unlock()

	err = this.getSession(this.svc, msg, resp)
	if err != nil {
		log.Errorf("client.VirtualIPConnect | Connect this.getSession err:%s", err.Error())
		return err
	}

	p := topics.NewMemProvider()
	log.Debugf("client.VirtualIPConnect | register ID:%s", string(this.svc.sess.ID()))
	topics.Register(this.svc.sess.ID(), p)

	this.svc.topicsMgr, err = topics.NewManager(this.svc.sess.ID())
	if err != nil {
		log.Errorf("client | Connect topics.NewManager err:%s", err.Error())
		return err
	}

	this.svc.csMgr.updateState(Ready)
	if err = this.svc.start(); err != nil {
		log.Errorf("client | Connect this.svc.start err:%s", err.Error())
		this.svc.stop()
		return err
	}

	this.svc.inStat.increment(int64(msg.Len()))
	this.svc.outStat.increment(int64(resp.Len()))

	return nil
}

// Publish sends a single MQTT PUBLISH message to the server. On completion, the
// supplied OnCompleteFunc is called. For QOS 0 messages, onComplete is called
// immediately after the message is sent to the outgoing buffer. For QOS 1 messages,
// onComplete is called when PUBACK is received. For QOS 2 messages, onComplete is
// called after the PUBCOMP message is received.
func (this *Client) Publish(msg *message.PublishMessage, onComplete OnCompleteFunc) error {
	return this.svc.publish(msg, onComplete)
}

// Subscribe sends a single SUBSCRIBE message to the server. The SUBSCRIBE message
// can contain multiple topics that the client wants to subscribe to. On completion,
// which is when the client receives a SUBACK messsage back from the server, the
// supplied onComplete funciton is called.
//
// When messages are sent to the client from the server that matches the topics the
// client subscribed to, the onPublish function is called to handle those messages.
// So in effect, the client can supply different onPublish functions for different
// topics.
func (this *Client) Subscribe(msg *message.SubscribeMessage, onComplete OnCompleteFunc, onPublish OnPublishFunc) error {
	return this.svc.subscribe(msg, onComplete, onPublish)
}

// Unsubscribe sends a single UNSUBSCRIBE message to the server. The UNSUBSCRIBE
// message can contain multiple topics that the client wants to unsubscribe. On
// completion, which is when the client receives a UNSUBACK message from the server,
// the supplied onComplete function is called. The client will no longer handle
// messages from the server for those unsubscribed topics.
func (this *Client) Unsubscribe(msg *message.UnsubscribeMessage, onComplete OnCompleteFunc) error {
	return this.svc.unsubscribe(msg, onComplete)
}

// Ping sends a single PINGREQ message to the server. PINGREQ/PINGRESP messages are
// mainly used by the client to keep a heartbeat to the server so the connection won't
// be dropped.
func (this *Client) Ping(onComplete OnCompleteFunc) error {
	return this.svc.ping(onComplete)
}

// Disconnect sends a single DISCONNECT message to the server. The client immediately
// terminates after the sending of the DISCONNECT message.
func (this *Client) Disconnect() {
	//msg := message.NewDisconnectMessage()
	this.svc.stop()
	this.svc.csMgr.updateState(Shutdown)
}

func (this *Client) getSession(svc *service, req *message.ConnectMessage, resp *message.ConnackMessage) error {
	//id := string(req.ClientId())
	svc.sess = &sessions.Session{}
	return svc.sess.Init(req)
}

func (this *Client) checkConfiguration() {
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
}

func (this *Client) WaitForStateChange(ctx context.Context, sourceState State) bool {
	ch := this.svc.csMgr.getNotifyChan()
	if this.svc.csMgr.getState() != sourceState {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-ch:
		return true
	}
}

// GetState returns the connectivity.State of ClientConn.
// This is an EXPERIMENTAL API.
func (this *Client) GetState() State {
	return this.svc.csMgr.getState()
}
