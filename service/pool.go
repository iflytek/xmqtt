package service

import (
	"xmqtt/utils/surgemq/message"
	"sync"
)

type PMPool struct {
	pool sync.Pool
}

func (p *PMPool)init() {
	p.pool.New = func() interface{} {
		msg := &message.PublishMessage{}
		msg.SetType(message.PUBLISH)
		return msg
	}
}

func (p *PMPool)PMPoolGet() *message.PublishMessage {
	return p.pool.Get().(*message.PublishMessage)
}

func (p *PMPool)PMPoolPut(b *message.PublishMessage) {
	b.ClearPacketId()
	p.pool.Put(b)
}

var MsgChan chan UploadMsg


