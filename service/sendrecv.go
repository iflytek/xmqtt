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
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"xmqtt/consts"
	"xmqtt/utils/cnt"
	"xmqtt/utils/log"
	"xmqtt/utils/recover"
	"xmqtt/utils/surgemq/message"
)

type netReader interface {
	io.Reader
	SetReadDeadline(t time.Time) error
}

type timeoutReader struct {
	d    int64 // 心跳时间，单位秒
	conn netReader
}

func (r *timeoutReader) Read(b []byte) (int, error) {
	heartbeat := time.Second * time.Duration(atomic.LoadInt64(&r.d))
	if err := r.conn.SetReadDeadline(time.Now().Add(heartbeat)); err != nil {
		return 0, err
	}
	return r.conn.Read(b)
}

// 更新心跳
func (this *service) updateHeartbeat(r *timeoutReader, keepAlive int32) {
	recover.PanicHandler()

	var newKeepAlive int32
	old := keepAlive
	for {
		select {
		case newKeepAlive = <-this.keepAliveCH:
			if atomic.CompareAndSwapInt32(&old, old, newKeepAlive) {
				log.Debugf("sendrecv.updateHeartbeat | productKey %s deviceName %s heartbeat old:%d new:%d", this.productKey, this.deviceName, old, newKeepAlive)
				temp := int64(newKeepAlive + 5)
				atomic.StoreInt64(&r.d, temp)
				log.Debugf("sendrecv.updateHeartbeat | productKey %s deviceName %s r.d:p", this.productKey, this.deviceName, r.d)
				heartbeat := time.Second * time.Duration(temp)
				if err := r.conn.SetReadDeadline(time.Now().Add(heartbeat)); err != nil {
					log.Errorf("sendrecv.updateHeartbeat | productKey %s deviceName %s SetReadDeadline err:%s", this.productKey, this.deviceName, err.Error())
				}
			}
		case <-this.done:
			return
		}
	}
}

// receiver() reads data from the network, and writes the data into the incoming buffer
func (this *service) receiver(server *Server) {
	defer func() {
		// Let's recover from panic
		recover.PanicHandler()
		this.wgStopped.Done()
		log.Debugf("sendrecv.receiver | productKey %s deviceName %s Stopping receiver", this.productKey, this.deviceName)
	}()

	log.Debugf("sendrecv.receiver | productKey %s deviceName %s Starting receiver", this.productKey, this.deviceName)

	this.wgStarted.Done()

	switch conn := this.conn.(type) {
	case net.Conn:
		log.Debugf("sendrecv.receiver | productKey %s deviceName %s setting read deadline to keepAlive %d(s)", this.productKey, this.deviceName, this.keepAlive)
		heartbeat := time.Second * time.Duration(this.keepAlive)
		if err := conn.SetReadDeadline(time.Now().Add(heartbeat)); err != nil {
			log.Errorf("sendrecv.receiver | productKey %s deviceName %s SetReadDeadline err:%s", this.productKey, this.deviceName, err.Error())
			return
		}

		// 延迟几秒关闭
		keepAlive := int64(this.keepAlive + 5)
		r := timeoutReader{
			d:    keepAlive,
			conn: conn,
		}

		// 开启智能心跳修改goroutine
		go this.updateHeartbeat(&r, (int32)(this.keepAlive))

		for {
			_, err := this.in.ReadFrom(&r)
			if err != nil {
				log.Errorf("sendrecv.receiver | productKey %s deviceName %s error reading from connection: %v", this.productKey, this.deviceName, err)
				key := strings.Join([]string{this.productKey, ":", this.deviceName}, "")
				svcOld, ok := server.svcMap.Load(key)
				if err != io.EOF {
					if ok {
						svc := svcOld.(*service)
						if strings.Contains(err.Error(), "connection reset by peer") {
							svc.reason = consts.LogReasonValConnectionResetByPeer
						}else {
							svc.reason = consts.LogReasonValKeepaliveTimeout
						}

						val := server.deleteMapAndLogout(svc)

						if strings.Contains(err.Error(), "connection reset by peer") {
							log.Errorf("deleteMapAndLogout | event: %s, reason: %s, err: %s", cnt.DeviceLogout, consts.LogReasonValConnectionResetByPeer, val.Error())
						} else {
							log.Errorf("deleteMapAndLogout | event: %s, reason: %s, err: %s", cnt.DeviceLogout, consts.LogReasonValKeepaliveTimeout, val.Error())
						}

						server.sessMgr.Del(svc.sess.ID())
						svc.newStopConnect()
						svc = nil
					}
					this.csMgr.updateState(TransientFailure)
				} else {
					if ok {
						svc := svcOld.(*service)
						svc.reason = consts.LogReasonValDeviceDisconnect
						val := server.deleteMapAndLogout(svc)
						if val != nil {
							log.Errorf("deleteMapAndLogout | event: %s, reason: %s, err: %s", cnt.DeviceLogout, consts.LogReasonValDeviceDisconnect, val.Error())
						}

						server.sessMgr.Del(svc.sess.ID())
						svc.newStopConnect()
						svc = nil
					}
					this.csMgr.updateState(Shutdown)
				}
				return
			} else {
				log.Debugf("sendrecv.receiver | productKey %s deviceName %s reading from connection", this.productKey, this.deviceName)
			}
		}
	default:
		log.Errorf("sendrecv.receiver | productKey %s deviceName %s %v", this.productKey, this.deviceName, ErrInvalidConnectionType)
	}
}

// sender() writes data from the outgoing buffer to the network
func (this *service) sender() {
	defer func() {
		// Let's recover from panic
		recover.PanicHandler()

		this.wgStopped.Done()

		log.Debugf("sendrecv.sender | productKey %s deviceName %s Stopping sender", this.productKey, this.deviceName)
	}()

	log.Debugf("sendrecv.sender | productKey %s deviceName %s Starting sender", this.productKey, this.deviceName)

	this.wgStarted.Done()

	switch conn := this.conn.(type) {
	case net.Conn:
		for {
			_, err := this.out.WriteTo(conn)
			if err != nil {
				log.Errorf("sendrecv.sender | productKey %s deviceName %s error writing data: %v", this.productKey, this.deviceName, err)
				if err != io.EOF {
					this.csMgr.updateState(TransientFailure)
				} else {
					this.csMgr.updateState(Shutdown)
				}
				return
			}
		}
	default:
		log.Errorf("sendrecv.sender | productKey %s deviceName %s Invalid connection type", this.productKey, this.deviceName)
	}
}

// peekMessageSize() reads, but not commits, enough bytes to determine the size of
// the next message and returns the type and size.
func (this *service) peekMessageSize() (message.MessageType, int, error) {
	var (
		b   []byte
		err error
		cnt int = 2
	)

	if this.in == nil {
		err = ErrBufferNotReady
		return 0, 0, err
	}

	// Let's read enough bytes to get the message header (msg type, remaining length)
	for {
		// If we have read 5 bytes and still not done, then there's a problem.
		if cnt > 5 {
			return 0, 0, fmt.Errorf("sendrecv/peekMessageSize: 4th byte of remaining length has continuation bit set")
		}

		// Peek cnt bytes from the input buffer.
		b, err = this.in.ReadWait(cnt)
		if err != nil {
			return 0, 0, err
		}

		// If not enough bytes are returned, then continue until there's enough.
		if len(b) < cnt {
			continue
		}

		// If we got enough bytes, then check the last byte to see if the continuation
		// bit is set. If so, increment cnt and continue peeking
		if b[cnt-1] >= 0x80 {
			cnt++
		} else {
			break
		}
	}

	// Get the remaining length of the message
	remlen, m := binary.Uvarint(b[1:])

	// Total message length is remlen + 1 (msg type) + m (remlen bytes)
	total := int(remlen) + 1 + m

	mtype := message.MessageType(b[0] >> 4)

	return mtype, total, err
}

// peekMessage() reads a message from the buffer, but the bytes are NOT committed.
// This means the buffer still thinks the bytes are not read yet.
func (this *service) peekMessage(mtype message.MessageType, total int) (message.Message, int, error) {
	var (
		b    []byte
		err  error
		i, n int
		msg  message.Message
	)

	if this.in == nil {
		return nil, 0, ErrBufferNotReady
	}

	// Peek until we get total bytes
	for i = 0; ; i++ {
		// Peek remlen bytes from the input buffer.
		b, err = this.in.ReadWait(total)
		if err != nil && err != ErrBufferInsufficientData {
			return nil, 0, err
		}

		// If not enough bytes are returned, then continue until there's enough.
		if len(b) >= total {
			break
		}
	}

	msg, err = mtype.New()
	if err != nil {
		return nil, 0, err
	}

	n, err = msg.Decode(b)
	return msg, n, err
}

// readMessage() reads and copies a message from the buffer. The buffer bytes are
// committed as a result of the read.
func (this *service) readMessage(mtype message.MessageType, total int) (message.Message, int, error) {
	var (
		b   []byte
		err error
		n   int
		msg message.Message
	)

	if this.in == nil {
		err = ErrBufferNotReady
		return nil, 0, err
	}

	if len(this.intmp) < total {
		this.intmp = make([]byte, total)
	}

	// Read until we get total bytes
	l := 0
	for l < total {
		n, err = this.in.Read(this.intmp[l:])
		l += n
		log.Debugf("sendrecv.readMessage | productKey %s deviceName %s read %d bytes, total %d", this.productKey, this.deviceName, n, l)
		if err != nil {
			return nil, 0, err
		}
	}

	b = this.intmp[:total]

	msg, err = mtype.New()
	if err != nil {
		return msg, 0, err
	}

	n, err = msg.Decode(b)
	return msg, n, err
}

// writeMessage() writes a message to the outgoing buffer
func (this *service) writeMessage(msg message.Message) (int, error) {
	var (
		l    int = msg.Len()
		m, n int
		err  error
		buf  []byte
		wrap bool
	)

	if this.out == nil {
		return 0, ErrBufferNotReady
	}

	// This is to serialize writes to the underlying buffer. Multiple goroutines could
	// potentially get here because of calling Publish() or Subscribe() or other
	// functions that will send messages. For example, if a message is received in
	// another connetion, and the message needs to be published to this client, then
	// the Publish() function is called, and at the same time, another client could
	// do exactly the same thing.
	//
	// Not an ideal fix though. If possible we should remove mutex and be lockfree.
	// Mainly because when there's a large number of goroutines that want to publish
	// to this client, then they will all block. However, this will do for now.
	//
	// FIXME: Try to find a better way than a mutex...if possible.
	this.wmu.Lock()
	defer this.wmu.Unlock()

	buf, wrap, err = this.out.WriteWait(l)
	if err != nil {
		return 0, err
	}

	if wrap {
		if len(this.outtmp) < l {
			this.outtmp = make([]byte, l)
		}

		n, err = msg.Encode(this.outtmp[0:])
		if err != nil {
			return 0, err
		}

		m, err = this.out.Write(this.outtmp[0:n])
		if err != nil {
			return m, err
		}
	} else {
		n, err = msg.Encode(buf[0:])
		if err != nil {
			return 0, err
		}

		m, err = this.out.WriteCommit(n)
		if err != nil {
			return 0, err
		}
	}

	this.outStat.increment(int64(m))

	return m, nil
}
