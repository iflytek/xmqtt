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
	"xmqtt/sessions"
	"xmqtt/utils/log"
	"xmqtt/utils/recover"
	"xmqtt/utils/surgemq/message"
	"io"
	"reflect"
	"strconv"
)

var (
	errDisconnect = errors.New("Disconnect")
)

// processor() reads messages from the incoming buffer and processes them
func (this *service) processor(server *Server) {
	defer func() {
		// Let's recover from panic
		recover.PanicHandler()
		this.wgStopped.Done()
		log.Debugf("process.processor | productKey %s deviceName %s Stopping processor", this.productKey, this.deviceName)
	}()

	log.Debugf("process.processor | productKey %s deviceName %s Starting processor", this.productKey, this.deviceName)
	this.wgStarted.Done()

	for {
		// Find out what message is next and the size of the message
		mtype, total, err := this.peekMessageSize()
		if err != nil {
			log.Errorf("process.processor | productKey %s deviceName %s Error peeking next message size: %v", this.productKey, this.deviceName, err)
			return
		}

		msg, n, err := this.peekMessage(mtype, total)
		if err != nil {
			log.Errorf("process.processor | productKey %s deviceName %s Error peeking next message: %v", this.productKey, this.deviceName, err)
			return
		}

		this.inStat.increment(int64(n))

		// Process the read message
		err = this.processIncoming(msg)
		if err != nil {
			if err != errDisconnect {
				log.Errorf("process.processor | productKey %s deviceName %s Error processing %s : %v", this.productKey, this.deviceName, msg.Name(), err)
			} else {
				log.Errorf("process.processor | productKey %s deviceName %s Error processing %s Disconnect : %v", this.productKey, this.deviceName, msg.Name(), err)
				return
			}
		}

		// We should commit the bytes in the buffer so we can move on
		_, err = this.in.ReadCommit(total)
		if err != nil {
			if err != io.EOF {
				log.Errorf("process.processor | productKey %s deviceName %s Error committing %d read bytes: %v", this.productKey, this.deviceName, total, err)
			}
			log.Errorf("process.processor | productKey %s deviceName %s Error committing %d read bytes EOF: %v", this.productKey, this.deviceName, total, err)
			return
		}

		// Check to see if done is closed, if so, exit
		if this.isDone() && this.in.Len() == 0 {
			return
		}
	}
}

func (this *service) processIncoming(msg message.Message) error {
	var err error = nil
	switch msg := msg.(type) {
	case *message.PublishMessage:
		// For PUBLISH message, we should figure out what QoS it is and process accordingly
		// If QoS == 0, we should just take the next step, no ack required 最多一次
		// If QoS == 1, we should send back PUBACK, then take the next step 至少一次
		// If QoS == 2, we need to put it in the ack queue, send back PUBREC 只有一次
		// 当前是处理两类topic(/sys/service/heartbeat/update、/sys/{productKey}/{deviceName}/thing/upload)上来的信息
		err = this.processPublish(msg)

	case *message.PubackMessage:
		// For PUBACK message, it means QoS 1, we should send to ack queue
		this.sess.Pub1ack.Ack(msg)
		this.processAcked(this.sess.Pub1ack)

	case *message.PubrecMessage: // QoS2特有：发布收到，第一步
		// For PUBREC message, it means QoS 2, we should send to ack queue, and send back PUBREL
		if err = this.sess.Pub2out.Ack(msg); err != nil {
			break
		}

		resp := message.NewPubrelMessage()
		resp.SetPacketId(msg.PacketId())
		_, err = this.sendData(resp)
	case *message.PubrelMessage: // QoS2特有：发布释放，第二步
		// For PUBREL message, it means QoS 2, we should send to ack queue, and send back PUBCOMP
		if err = this.sess.Pub2in.Ack(msg); err != nil {
			break
		}

		this.processAcked(this.sess.Pub2in)

		resp := message.NewPubcompMessage()
		resp.SetPacketId(msg.PacketId())
		_, err = this.sendData(resp)

	case *message.PubcompMessage: // QoS2特有：发布完成，第三步
		// For PUBCOMP message, it means QoS 2, we should send to ack queue
		if err = this.sess.Pub2out.Ack(msg); err != nil {
			break
		}
		this.processAcked(this.sess.Pub2out)

	case *message.SubscribeMessage:
		// For SUBSCRIBE message, we should add subscriber, then send back SUBACK
		return this.processSubscribe(msg)

	case *message.SubackMessage:
		// For SUBACK message, we should send to ack queue
		this.sess.Suback.Ack(msg)
		this.processAcked(this.sess.Suback)

	case *message.UnsubscribeMessage:
		// For UNSUBSCRIBE message, we should remove subscriber, then send back UNSUBACK
		return this.processUnsubscribe(msg)

	case *message.UnsubackMessage:
		// For UNSUBACK message, we should send to ack queue
		this.sess.Unsuback.Ack(msg)
		this.processAcked(this.sess.Unsuback)

	case *message.PingreqMessage:
		// For PINGREQ message, we should send back PINGRESP
		resp := message.NewPingrespMessage()
		_, err = this.sendData(resp)

	case *message.PingrespMessage:
		this.sess.Pingack.Ack(msg)
		this.processAcked(this.sess.Pingack)

	case *message.DisconnectMessage:
		// For DISCONNECT message, we should quit
		this.sess.Cmsg.SetWillFlag(false)
		return errDisconnect

	default:
		return fmt.Errorf("%s invalid message type %s.", this.productKey+":"+this.deviceName, msg.Name())
	}

	if err != nil {
		log.Debugf("process.processor | productKey %s deviceName %s Error processing acked message: %v", this.productKey, this.deviceName, err)
	}

	return err
}

func (this *service) processAcked(ackq *sessions.Ackqueue) {
	for _, ackmsg := range ackq.Acked() {
		// Let's get the messages from the saved message byte slices.

		msg, err := ackmsg.Mtype.New()
		if err != nil {
			log.Errorf("process.processAcked | productKey %s deviceName %s Unable to creating new %s message: %v", this.productKey, this.deviceName, ackmsg.Mtype, err)
			continue
		}

		if _, err := msg.Decode(ackmsg.Msgbuf); err != nil {
			log.Errorf("process.processAcked | productKey %s deviceName %s Unable to decode %s message: %v", this.productKey, this.deviceName, ackmsg.Mtype, err)
			continue
		}

		ack, err := ackmsg.State.New()
		if err != nil {
			log.Errorf("process.processAcked | productKey %s deviceName %s Unable to creating new %s message: %v", this.productKey, this.deviceName, ackmsg.State, err)
			continue
		}

		if _, err := ack.Decode(ackmsg.Ackbuf); err != nil {
			log.Errorf("process.processAcked | productKey %s deviceName %s Unable to decode %s message: %v", this.productKey, this.deviceName, ackmsg.State, err)
			continue
		}

		log.Infof("process.processAcked | (%s) Processing acked message: %v", this.productKey+":"+this.deviceName, ack)

		// - PUBACK if it's QoS 1 message. This is on the client side.
		// - PUBREL if it's QoS 2 message. This is on the server side.
		// - PUBCOMP if it's QoS 2 message. This is on the client side.
		// - SUBACK if it's a subscribe message. This is on the client side.
		// - UNSUBACK if it's a unsubscribe message. This is on the client side.
		switch ackmsg.State {
		case message.PUBREL:
			// If ack is PUBREL, that means the QoS 2 message sent by a remote client is
			// releassed, so let's publish it to other subscribers.
			if err = this.onPublish(msg.(*message.PublishMessage)); err != nil {
				log.Errorf("process.processAcked | productKey %s deviceName %s Error processing acked %s message: %v", this.productKey, this.deviceName, ackmsg.Mtype, err)
			}

		case message.PUBACK, message.PUBCOMP, message.SUBACK, message.UNSUBACK, message.PINGRESP:
			log.Debugf("process.processAcked | productKey %s deviceName %s process/processAcked: %s", this.productKey, this.deviceName, ack)
			// If ack is PUBACK, that means the QoS 1 message sent by this service got
			// ack'ed. There's nothing to do other than calling onComplete() below.

			// If ack is PUBCOMP, that means the QoS 2 message sent by this service got
			// ack'ed. There's nothing to do other than calling onComplete() below.

			// If ack is SUBACK, that means the SUBSCRIBE message sent by this service
			// got ack'ed. There's nothing to do other than calling onComplete() below.

			// If ack is UNSUBACK, that means the SUBSCRIBE message sent by this service
			// got ack'ed. There's nothing to do other than calling onComplete() below.

			// If ack is PINGRESP, that means the PINGREQ message sent by this service
			// got ack'ed. There's nothing to do other than calling onComplete() below.

			err = nil

		default:
			log.Errorf("process.processAcked | productKey %s deviceName %s Invalid ack message type %s.", this.productKey, this.deviceName, ackmsg.State)
			continue
		}

		// Call the registered onComplete function
		if ackmsg.OnComplete != nil {
			onComplete, ok := ackmsg.OnComplete.(OnCompleteFunc)
			if !ok {
				log.Errorf("process.processAcked | productKey %s deviceName %s Error type asserting onComplete function: %v", this.productKey, this.deviceName, reflect.TypeOf(ackmsg.OnComplete))
			} else if onComplete != nil {
				if err := onComplete(msg, ack, nil); err != nil {
					log.Errorf("process.processAcked | productKey %s deviceName %s Error running onComplete(): %v", this.productKey, this.deviceName, err)
				}
			}
		}
		ackq.BytesPool.Put(ackmsg.Ackbuf)
		ackq.BytesPool.Put(ackmsg.Msgbuf)
		ackmsg.Ackbuf = nil
		ackmsg.Msgbuf = nil
		ackmsg = nil
		msg = nil
		ack = nil
	}
}

// For PUBLISH message, we should figure out what QoS it is and process accordingly
// If QoS == 0, we should just take the next step, no ack required
// If QoS == 1, we should send back PUBACK, then take the next step
// If QoS == 2, we need to put it in the ack queue, send back PUBREC
func (this *service) processPublish(msg *message.PublishMessage) error {

	if string(msg.Topic()) == "/sys/service/heartbeat/update" {
		payload := msg.Payload()
		heartbeat, err := strconv.Atoi(string(payload))
		if err != nil {
			log.Warnf("processPublish | productKey %s deviceName %s update heartbeat err:%s", this.productKey, this.deviceName, err.Error())
			return err
		}

		// 心跳时间间隔：[1, 2*3600]
		if heartbeat < 1 {
			heartbeat = 1
		} else if heartbeat > 2*3600 {
			heartbeat = 2 * 3600
		}

		// 更新心跳
		log.Debugf("processPublish | productKey %s deviceName %s update heartbeat new:%d", this.productKey, this.deviceName, heartbeat)

		// 注意阻塞
		this.keepAliveCH <- int32(heartbeat)

		// 只支持Qos == 1的心跳更新响应
		if msg.QoS() == message.QosAtLeastOnce {
			resp := message.NewPubackMessage()
			resp.SetPacketId(msg.PacketId())

			if _, err := this.sendData(resp); err != nil {
				return err
			}
		}
		return nil
	}

	switch msg.QoS() {
	case message.QosAtLeastOnce:
		resp := message.NewPubackMessage()
		resp.SetPacketId(msg.PacketId())

		if _, err := this.sendData(resp); err != nil {
			return err
		}

		select {
			case MsgChan <- UploadMsg {
				Msg:        msg,
				DeviceName: this.productKey,
				ProductKey: this.deviceName,
			}:
				log.Infof("write data to channel success, dName: %s, pKey: %s", this.deviceName, this.productKey)
			default:
				log.Infof("channel is full, dName: %s, pKey: %s", this.deviceName, this.productKey)
		}

		return nil
	}

	return fmt.Errorf("productKey %s deviceName %s invalid message QoS %d.", this.productKey, this.deviceName, msg.QoS())
}


// For SUBSCRIBE message, we should add subscriber, then send back SUBACK
func (this *service) processSubscribe(msg *message.SubscribeMessage) error {
	resp := message.NewSubackMessage()
	resp.SetPacketId(msg.PacketId())

	// Subscribe to the different topics
	var retcodes []byte

	topics := msg.Topics()
	qos := msg.Qos()

	this.rmsgs = this.rmsgs[0:0]

	// 忽略topic订阅关系，仅通过productKey + deviceName 找到Session发送
	//for i, t := range topics {
	for i := range topics {
		//	rqos, err := this.topicsMgr.Subscribe(t, qos[i], &this.onpub)
		//if err != nil {
		//	return err
		//}
		//this.sess.AddTopic(string(t), qos[i])

		//retcodes = append(retcodes, rqos)
		retcodes = append(retcodes, qos[i])

		// yeah I am not checking errors here. If there's an error we don't want the
		// subscription to stop, just let it go.
		//this.topicsMgr.Retained(t, &this.rmsgs)
		//log.Infof("process.processSubscribe | (%s) topic = %s, retained count = %d", this.productKey+":"+this.deviceName, string(t), len(this.rmsgs))
	}

	if err := resp.AddReturnCodes(retcodes); err != nil {
		return err
	}

	if _, err := this.sendData(resp); err != nil {
		return err
	}

	return nil
}

// For UNSUBSCRIBE message, we should remove the subscriber, and send back UNSUBACK
func (this *service) processUnsubscribe(msg *message.UnsubscribeMessage) error {
	resp := message.NewUnsubackMessage()
	resp.SetPacketId(msg.PacketId())

	_, err := this.sendData(resp)
	return err
}

// onPublish() is called when the server receives a PUBLISH message AND have completed
// the ack cycle. This method will get the list of subscribers based on the publish
// topic, and publishes the message to the list of subscribers.
func (this *service) onPublish(msg *message.PublishMessage) error {
	log.Debugf("service.onPublish | productKey %s deviceName %s len(msg):%d\n", this.productKey, this.deviceName, len(msg.Payload()))
	return nil
}
