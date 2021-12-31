package service

import (
	"io"
	"time"

	"git.iflytek.com/HY_XIoT/core/utils/log"
	"git.iflytek.com/HY_XIoT/core/xfyun/surgemq/message"
)

func (this *service) keepaliver() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("(%s) Recovering from panic: %v", this.cid(), r)
		}
		this.wgStopped.Done()

		log.Debugf("(%s) Stopping keepaliver", this.cid())
	}()

	log.Debugf("(%s) Starting keepaliver", this.cid())

	this.wgStarted.Done()

	// 启动定时器，定时发送keepalive心跳，
	// 目前心跳为定时发送，不会因为收到消息而改变心跳频率

	ticker := time.NewTicker(time.Second * time.Duration(this.keepAlive))
	for {
		select {

		case <-ticker.C:
			// 发送ping
			req := message.NewPingreqMessage()
			_, err := this.writeMessage(req)
			log.Debugf("(%s) send keepalive", this.cid())
			if err != nil {
				log.Errorf("%s send keepalive err:%s", this.cid(), err.Error())
				if err != io.EOF {
					this.csMgr.updateState(TransientFailure)
				} else {
					this.csMgr.updateState(Shutdown)
				}
				return
			}
		case <-this.done:
			return
		}
	}
}
