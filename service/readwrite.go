package service

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"xmqtt/consts"
	"xmqtt/utils/cnt"
	"xmqtt/utils/log"
	"xmqtt/utils/recover"
	"xmqtt/utils/surgemq/message"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	defaultReadSize = 1024
	defaultWriteSize2K = 2 << 10
	defaultWriteSize4K = 4 << 10
	defaultWriteSize8K = 8 << 10

)

var ErrMessageTooLarge = errors.New("mqtt: message size exceeds 256K")

// 接受数据
func (this *service) receiverData(server *Server) {
	defer recover.PanicHandler()
	switch conn := this.conn.(type) {
	case net.Conn:
		heartbeat := time.Second * time.Duration(this.keepAlive)
		if err := conn.SetReadDeadline(time.Now().Add(heartbeat)); err != nil {
			log.Errorf("readwrite.receiverData | productKey %s deviceName %s SetReadDeadline err:%s", this.productKey, this.deviceName, err.Error())
			return
		}

		keepAlive := int64(this.keepAlive + 5)
		r := timeoutReader{
			d:    keepAlive,
			conn: conn,
		}

		go this.updateHeartbeat(&r, (int32)(this.keepAlive))
		reader := bufio.NewReaderSize(&r, defaultReadSize)
		for {

			mType, total, err := this.getPeekMessageSize(reader)
			if err != nil {
				this.exceptionDisconnect(server, err)
				return
			}

			if total > server.MaxBufferSize {
				log.Warnf("readwrite.receiverData | productKey %s deviceName %s err:%s", this.productKey, this.deviceName, ErrMessageTooLarge)
				this.exceptionDisconnect(server, err)
				return
			}

			msg, _, err := this.getPeekMessage(reader, mType, total)
			if err != nil {
				this.exceptionDisconnect(server, err)
				return
			}

			err = this.processIncoming(msg)
			if err != nil {
				this.exceptionDisconnect(server, err)
				return
			}

			if this.isDone() {
				return
			}
		}
	default:
		log.Errorf("readwrite.receiverData | productKey %s deviceName %s %v", this.productKey, this.deviceName, ErrInvalidConnectionType)
	}
}

// 异常断开
func (this *service) exceptionDisconnect(server *Server, err error) {
	//设备互踢、服务端主动断开、Ack超时、心跳超时、客户端主动关闭连接等场景都会执行
	log.Warnf("readwrite.exceptionDisconnect | productKey %s deviceName %s error reading from connection: %v", this.productKey, this.deviceName, err)
	key := strings.Join([]string{this.productKey, ":", this.deviceName}, "")
	svcOld, ok := server.svcMap.Load(key)
	if err != io.EOF {
		if ok {
			//删除map并设备下线
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
	} else {
		if ok {
			svc := svcOld.(*service)
			svc.reason = consts.LogReasonValDeviceDisconnect

			val := server.deleteMapAndLogout(svc)
			if val != nil{
				log.Errorf("deleteMapAndLogout | event: %s, reason: %s, err: %s", cnt.DeviceLogout, consts.LogReasonValDeviceDisconnect, val.Error())
			}

			server.sessMgr.Del(svc.sess.ID())
			svc.newStopConnect()
			svc = nil
		}
	}
}

// 获取固定包头及总长度
func (this *service) getPeekMessageSize(br *bufio.Reader) (message.MessageType, int, error) {
	var (
		b   []byte
		err error
		cnt = 2
	)

	// 读取足够的字节来获取消息头（msg类型，剩余长度）
	for {
		// 如果我们已经读取了5个字节但仍然没有完成，则会出现问题。
		if cnt > 5 {
			return 0, 0, fmt.Errorf("readwrite/getPeekMessageSize: 4th byte of remaining length has continuation bit set")
		}

		// 获取当前缓冲区内接下来的n个byte，但是不移动指针
		b, err = br.Peek(cnt)
		if err != nil {
			return 0, 0, err
		}

		// 如果没有返回足够的字节，则继续，直到足够。
		if len(b) < cnt {
			continue
		}

		// 如果我们得到足够的字节，那么检查最后一个字节以查看是否设置了连续位。如果是这样，增加cnt并继续
		if b[cnt-1] >= 0x80 {
			cnt++
		} else {
			break
		}
	}

	// 获取消息的剩余长度
	remlen, m := binary.Uvarint(b[1:])

	// 消息总长度为remlen + 1（msg类型）+ m（remlen字节）
	total := int(remlen) + 1 + m

	// 固定报头的控制报文类型
	mtype := message.MessageType(b[0] >> 4)
	return mtype, total, err
}

// 读取包及所有字节total
func (this *service) getPeekMessage(br *bufio.Reader, mtype message.MessageType, total int) (message.Message, int, error) {
	var (
		b   []byte
		err error
		//i   int
		n   int
		msg message.Message
	)

	b = make([]byte, total)
	m, err := io.ReadFull(br, b)
	if err != nil {
		log.Errorf("productKey %s deviceName %s total %d err %s", this.productKey, this.deviceName, m, err.Error())
	}

	//// 直到我们得到总字节数
	//for i = 0; ; i++ {
	//	// 从输入缓冲区中查看remlen字节。
	//	b, err = br.Peek(total)
	//	if err != nil && err != bufio.ErrBufferFull {
	//		return nil, 0, err
	//	}
	//
	//	// 如果没有返回足够的字节，则继续，直到足够。
	//	if len(b) >= total {
	//		log.Errorf("productKey %s deviceName %s x %d total %d mtype %v", this.productKey, this.deviceName, len(b), total, mtype)
	//		break
	//	}
	//}

	msg, err = mtype.New()
	if err != nil {
		return nil, 0, err
	}

	n, err = msg.Decode(b)
	return msg, n, err
}

// 发送数据
func (this *service) sendData(msg message.Message) (int, error) {
	baseTime := time.Now()
	defer func() {
		sub := time.Since(baseTime).Nanoseconds()
		if sub > 1000000000 {
			log.Debugf("readwrite.sendData|ts %d", sub)
		}
		recover.PanicHandler()
	}()

	var (
		l   = msg.Len()
		err error
		buf []byte
	)

	buf = make([]byte, l)
	n, err := msg.Encode(buf[0:])
	if err != nil {
		return n, err
	}

	buf = buf[0:n]
	switch conn := this.conn.(type) {
	case net.Conn:
			//write := bufio.NewWriterSize(conn, n)
			//_, err := write.Write(buf)
			//if err != nil {
			//	log.Errorf("readwrite.sendData | productKey %s deviceName %s messageType %s write error %v", this.productKey, this.deviceName, msg.Type().String())
			//	return 0, err
			//}
			//
			//err = write.Flush()
			//if err != nil{
			//	log.Errorf("readwrite.sendData flush | productKey %s deviceName %s messageType %s write error %v", this.productKey, this.deviceName, msg.Type().String())
			//	return 0, err
			//}
			var size int
			if n <= defaultWriteSize2K{
				size = defaultWriteSize2K

			} else if defaultWriteSize2K < n && n <= defaultWriteSize4K {
				size = defaultWriteSize4K

			}else if defaultWriteSize4K < n && n <= defaultWriteSize8K {
				size = defaultWriteSize8K

			}else{
				size = n
			}

			w := newBufioWriterSize(conn, size)
			_, err := w.Write(buf)
			if err != nil {
				log.Errorf("readwrite.sendData | productKey %s deviceName %s messageType %s write error %v", this.productKey, this.deviceName, msg.Type().String())
				return 0, err
			}

			err = w.Flush()
			if err != nil{
				log.Errorf("readwrite.sendData flush | productKey %s deviceName %s messageType %s write error %v", this.productKey, this.deviceName, msg.Type().String())
				return 0, err
			}

		switch size {
		case defaultWriteSize2K, defaultWriteSize4K, defaultWriteSize8K:
			putBufioWriter(w)
		default:
			w.Reset(nil)
		}

	default:
		log.Errorf("readwrite.sendData | productKey %s deviceName %s messageType %s Invalid connection type", this.productKey, this.deviceName, msg.Type().String())
	}
	return 0, nil
}

var (
	bufioWriter2kPool sync.Pool
	bufioWriter4kPool sync.Pool
	bufioWriter8kPool sync.Pool
)

func newBufioWriterSize(w io.Writer, size int) *bufio.Writer {
	pool := bufioWriterPool(size)
	if pool != nil {
		if v := pool.Get(); v != nil {
			bw := v.(*bufio.Writer)
			bw.Reset(w)
			return bw
		}
	}
	return bufio.NewWriterSize(w, size)
}

func putBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	if pool := bufioWriterPool(bw.Available()); pool != nil {
		pool.Put(bw)
	}
}

func bufioWriterPool(size int) *sync.Pool {
	switch size {
	case 2 << 10:
		return &bufioWriter2kPool
	case 4 << 10:
		return &bufioWriter4kPool
	}
	return nil
}
