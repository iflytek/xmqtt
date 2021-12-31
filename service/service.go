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
	"bufio"
	"errors"
	"fmt"
	"xmqtt/sessions"
	"xmqtt/utils/log"
	"xmqtt/utils/recover"
	"xmqtt/utils/surgemq/message"
	"github.com/jinzhu/copier"
	"github.com/rfyiamcool/go-timewheel"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type (
	OnCompleteFunc func(msg, ack message.Message, err error) error
	OnPublishFunc  func(msg *message.PublishMessage) error
)

type stat struct {
	bytes int64
	msgs  int64
}

type UploadMsg struct {
	Msg *message.PublishMessage
	DeviceName string
	ProductKey string
}

func (this *stat) increment(n int64) {
	atomic.AddInt64(&this.bytes, n)
	atomic.AddInt64(&this.msgs, 1)
}

var (
	gsvcid uint64 = 0
)

type service struct {
	// The ID of this service, it's not related to the Client ID, just a number that's
	// incremented for every new service.
	id uint64

	// Is this a client or server. It's set by either Connect (client) or
	// HandleConnection (server).
	client bool

	// remote client address
	clientAddr string

	// product Key
	productKey string

	// client name
	deviceName string

	// xmqtt ID
	nodeID string

	//signature algorithm type
	algorithm string

	// Time stamp of client authentication
	timeStamp int64

	// MqttPassword signature string encoded by Base64 algorithm
	signature string

	// The number of seconds to keep the connection live if there's no data.
	// If not set then default to 5 mins.
	keepAlive int

	// 更新心跳
	keepAliveCH chan int32

	// The number of seconds to wait for the CONNACK message before disconnecting.
	// If not set then default to 2 seconds.
	connectTimeout int

	// The number of seconds to wait for any ACK messages before failing.
	// If not set then default to 20 seconds.
	ackTimeout int

	// The number of times to retry sending a packet if ACK is not received.
	// If no set then default to 3 retries.
	timeoutRetries int

	// Network connection for this service
	conn io.Closer

	// Session manager for tracking all the clients
	//sessMgr *sessions.Manager

	// Topics manager for all the client subscriptions
	//topicsMgr *topics.Manager

	// sess is the session object for this MQTT session. It keeps track session variables
	// such as ClientId, KeepAlive, Username, etc
	sess *sessions.Session

	// Wait for the various goroutines to finish starting and stopping
	wgStarted sync.WaitGroup
	wgStopped sync.WaitGroup

	// writeMessage mutex - serializes writes to the outgoing buffer.
	wmu sync.Mutex
	closeLock sync.Mutex

	// Whether this is service is closed or not.
	//closed int64

	// Quit signal for determining when this service should end. If channel is closed,
	// then exit.
	done chan struct{}

	// Incoming data buffer. Bytes are read from the connection and put in here.
	in *buffer

	// Outgoing data buffer. Bytes written here are in turn written out to the connection.
	out *buffer

	// onpub is the method that gets added to the topic subscribers list by the
	// processSubscribe() method. When the server finishes the ack cycle for a
	// PUBLISH message, it will call the subscriber, which is this method.
	//
	// For the server, when this method is called, it means there's a message that
	// should be published to the client on the other end of this connection. So we
	// will call publish() to send the message.
	onPub    OnPublishFunc
	inStat  stat
	outStat stat

	intmp  []byte
	outtmp []byte

	subs  []interface{}
	qoss  []byte
	rmsgs []*message.PublishMessage

	csMgr *connectivityStateManager

	//登录状态值
	loginValue string

	clientId string

	accountId string

	tw *timewheel.TimeWheel
	Map sync.Map
	pm PMPool

	newWriter *bufio.Writer

	reason string
}

func (this *service) start(server *Server) error {

	if !this.client {
		// Creat the onPublishFunc so it can be used for published messages
		this.onPub = func(msg *message.PublishMessage) error {
			if err := this.publish( msg, nil); err != nil {
				log.Errorf("service.start | productKey %s deviceName %s onPublish publishing message err:%v", this.productKey, this.deviceName, err)
				return err
			}
			return nil
		}
	}

	go this.receiverData(server)
	return nil
}

// FIXME: The order of closing here causes panic sometimes. For example, if receiver
// calls this, and closes the buffers, somehow it causes buffer.go:476 to panid.
func (this *service) stopConnect() {
	defer recover.PanicHandler()

	// Close quit channel, effectively telling all the goroutines it's time to quit
	if this.done != nil {
		log.Debugf("server.stop | productKey %s deviceName %s closing this.done", this.productKey, this.deviceName)
		close(this.done)
	}

	// Close the network connection
	if this.conn != nil {
		log.Debugf("server.stop | productKey %s deviceName %s closing this.conn", this.productKey, this.deviceName)
		this.conn.Close()
	}

	this.in.Close()
	this.out.Close()

	// Wait for all the goroutines to stop.
	this.wgStopped.Wait()

	if this.keepAliveCH != nil {
		close(this.keepAliveCH)
	}

	this.conn = nil
	this.in = nil
	this.out = nil
}

// 去除in、out buffer关闭连接方法
func (this *service) newStopConnect() {
	defer recover.PanicHandler()
	// Close quit channel, effectively telling all the goroutines it's time to quit
	if this.done != nil {
		log.Debugf("server.stop | productKey %s deviceName %s closing this.done", this.productKey, this.deviceName)
		close(this.done)
	}

	// Close the network connection
	if this.conn != nil {
		log.Debugf("server.stop | productKey %s deviceName %s closing this.conn", this.productKey, this.deviceName)
		this.conn.Close()
	}

	if this.keepAliveCH != nil {
		close(this.keepAliveCH)
	}

	this.conn = nil
	this.closeLock.Lock()
	this.sess.Stop()
	this.sess = nil
	this.closeLock.Unlock()
}

func (this *service) onComplete(msg, ack message.Message, err error) (ERR error) {
	switch ack.Type() {
	case message.PUBACK:
		packetId := msg.PacketId()
		ackPacketId := ack.PacketId()
		value, ok := this.Map.Load(ackPacketId)
		if ok {
			if task, ok := value.(*timewheel.Task); ok{
				this.tw.Remove(task)
			}
			this.Map.Delete(ackPacketId)

		}else{
			log.Errorf("service.onComplete | has received the publish message's ack:%s from productKey %s deviceName %s msgPacketId %d ackPacketId %d, but cannot find timer", message.PUBACK.String(), this.productKey, this.deviceName, packetId, ackPacketId)
		}

		log.Debugf("service.onComplete | has received the publish message's ack:%s from productKey %s deviceName %s msgPacketId %d ackPacketId %d", message.PUBACK.String(), this.productKey, this.deviceName, packetId, ackPacketId)
		return
	case message.PUBCOMP, message.SUBACK, message.UNSUBACK, message.PINGRESP:
		log.Warnf("service.onComplete | current system does not support these ack types")
		ERR = errors.New("does not support these ack types")
		return
	default:
		log.Errorf("service.onComplete | (%s) Invalid ack message type %s", this.productKey+":"+this.deviceName, ack.Type())
		ERR = errors.New("invalid message for acking")
		return
	}
}

type Data struct {
	Ret int `json:"ret"`
	Msg string `json:"msg"`
	Identifer string `json:"identifer"`
	NetworkTimeout bool `json:"networkTimeout"`
}

func (this *service) publish(msg *message.PublishMessage, onComplete OnCompleteFunc) error {
	_, err := this.sendData(msg)
	if err != nil {
		return fmt.Errorf("(%s) Error sending %s message: %v", this.productKey+":"+this.deviceName, msg.Name(), err)
	}

	switch msg.QoS() {
	case message.QosAtMostOnce:
		if onComplete != nil {
			return onComplete(msg,nil, nil)
		}
		return nil

	case message.QosAtLeastOnce:
		copyMsg := this.pm.PMPoolGet()
		copier.Copy(copyMsg, msg)

		task := this.tw.Add(time.Duration(this.ackTimeout)*time.Second, func() {
			defer this.pm.PMPoolPut(copyMsg)
			log.Debugf("ackTimeout: %d, productKey: %s, deviceName: %s, playload: %s", this.ackTimeout, this.productKey, this.deviceName, string(copyMsg.Payload()))
			val, ok := this.Map.Load(copyMsg.PacketId())
			if ok {
				t := val.(*timewheel.Task)
				if t != nil{
					t = nil
				}
			}
			this.Map.Delete(copyMsg.PacketId())
		})

		this.Map.Store(msg.PacketId(), task)
		this.closeLock.Lock()
		if this.sess != nil && this.sess.Pub1ack != nil {
			err := this.sess.Pub1ack.Wait(msg, onComplete)
			this.closeLock.Unlock()
			return err
		}
		this.closeLock.Unlock()
	case message.QosExactlyOnce:
		return this.sess.Pub2out.Wait(msg, onComplete)
	}
	return nil
}

var ackTickerMap sync.Map

//基于productKey和deviceName存储定时任务
func (this *service) addTicker(productKey, deviceName string, timeout int) *time.Ticker {
	baseTime := time.Now()
	defer func() {
		sub := time.Since(baseTime).Nanoseconds()
		log.Warnf("service.addTicker|ts %d", sub)
	}()
	var t *time.Ticker
	s := strings.Join([]string{productKey, deviceName}, ":")
	if v, ok := ackTickerMap.Load(s); ok {
		if ticker, ok := v.(*time.Ticker); ok {
			ticker.Stop()
			t = time.NewTicker(time.Duration(timeout) * time.Second)
		}
	} else {
		t = time.NewTicker(time.Duration(timeout) * time.Second)
	}
	ackTickerMap.Store(s, t)
	return t
}

//基于productKey和deviceName移除定时任务
func (this *service) removeTicker(productKey, deviceName string) {
	s := strings.Join([]string{productKey, deviceName}, ":")
	if v, ok := ackTickerMap.Load(s); ok {
		if ticker, ok := v.(*time.Ticker); ok {
			ticker.Stop()
			ackTickerMap.Delete(s)
		}
	}
}

func (this *service) subscribe(msg *message.SubscribeMessage, onComplete OnCompleteFunc, onPublish OnPublishFunc) error {
	if onPublish == nil {
		return fmt.Errorf("onPublish function is nil. No need to subscribe.")
	}

	_, err := this.sendData(msg)
	if err != nil {
		return fmt.Errorf("(%s) Error sending %s message: %v", this.productKey+":"+this.deviceName, msg.Name(), err)
	}

	var onc OnCompleteFunc = func(msg, ack message.Message, err error) error {
		onComplete := onComplete
		//	onPublish := onPublish

		if err != nil {
			if onComplete != nil {
				return onComplete(msg, ack, err)
			}
			return err
		}
		sub, ok := msg.(*message.SubscribeMessage)
		if !ok {
			if onComplete != nil {
				return onComplete(msg, ack, fmt.Errorf("Invalid SubscribeMessage received"))
			}
			return nil
		}

		suback, ok := ack.(*message.SubackMessage)
		if !ok {
			if onComplete != nil {
				return onComplete(msg, ack, fmt.Errorf("Invalid SubackMessage received"))
			}
			return nil
		}

		if sub.PacketId() != suback.PacketId() {
			if onComplete != nil {
				return onComplete(msg, ack, fmt.Errorf("Sub and Suback packet ID not the same. %d != %d.", sub.PacketId(), suback.PacketId()))
			}
			return nil
		}

		retcodes := suback.ReturnCodes()
		topics := sub.Topics()

		if len(topics) != len(retcodes) {
			if onComplete != nil {
				return onComplete(msg, ack, fmt.Errorf("Incorrect number of return codes received. Expecting %d, got %d.", len(topics), len(retcodes)))
			}
			return nil
		}

		var err2 error = nil

		for i, t := range topics {
			c := retcodes[i]

			if c == message.QosFailure {
				err2 = fmt.Errorf("Failed to subscribe to '%s'\n%v", string(t), err2)
			}
		}

		if onComplete != nil {
			return onComplete(msg, ack, err2)
		}

		return err2
	}

	return this.sess.Suback.Wait(msg, onc)
}

func (this *service) unsubscribe(msg *message.UnsubscribeMessage, onComplete OnCompleteFunc) error {
	_, err := this.sendData(msg)
	if err != nil {
		return fmt.Errorf("(%s) Error sending %s message: %v", this.productKey+":"+this.deviceName, msg.Name(), err)
	}

	var onc OnCompleteFunc = func(msg, ack message.Message, err error) error {
		onComplete := onComplete

		if err != nil {
			if onComplete != nil {
				return onComplete(msg, ack, err)
			}
			return err
		}

		unsub, ok := msg.(*message.UnsubscribeMessage)
		if !ok {
			if onComplete != nil {
				return onComplete(msg, ack, fmt.Errorf("Invalid UnsubscribeMessage received"))
			}
			return nil
		}

		unsuback, ok := ack.(*message.UnsubackMessage)
		if !ok {
			if onComplete != nil {
				return onComplete(msg, ack, fmt.Errorf("Invalid UnsubackMessage received"))
			}
			return nil
		}

		if unsub.PacketId() != unsuback.PacketId() {
			if onComplete != nil {
				return onComplete(msg, ack, fmt.Errorf("Unsub and Unsuback packet ID not the same. %d != %d.", unsub.PacketId(), unsuback.PacketId()))
			}
			return nil
		}

		var err2 error = nil

		//for _, tb := range unsub.Topics() {
		// Remove all subscribers, which basically it's just this client, since
		// each client has it's own topic tree.
		/*
			err := this.topicsMgr.Unsubscribe(tb, nil)
			if err != nil {
				err2 = fmt.Errorf("%v\n%v", err2, err)
			}

			this.sess.RemoveTopic(string(tb))
		*/
		//}

		if onComplete != nil {
			return onComplete(msg, ack, err2)
		}

		return err2
	}

	return this.sess.Unsuback.Wait(msg, onc)
}

func (this *service) ping(onComplete OnCompleteFunc) error {
	msg := message.NewPingreqMessage()

	_, err := this.sendData(msg)
	if err != nil {
		return fmt.Errorf("(%s) Error sending %s message: %v", this.productKey+":"+this.deviceName, msg.Name(), err)
	}

	return this.sess.Pingack.Wait(msg, onComplete)
}

func (this *service) isDone() bool {
	select {
	case <-this.done:
		return true

	default:
	}

	return false
}

func (this *service) cid() string {
	return fmt.Sprintf("%d/%s", this.id, this.sess.ID())
}
