package service

import (
	"fmt"
	"xmqtt/utils/log"
	"xmqtt/utils/surgemq/message"
	"strings"
)

func (this *Server) PushMsg() {
	num := 1
	for i := 0; i < num; i ++ {
		go func() {
			for{
				select{
				case msg := <- MsgChan:
					topic := msg.Msg.Topic()
					path := strings.Split(string(topic), "/")
					if len(path) < 2 {
						log.Errorf("path invalid, key: %s, path: %+v", topic, path)
						return
					}

					pKey, dName := path[len(path)-2],path[len(path)-1]
					key := strings.Join(path[len(path)-2:], ":")
					svc, ok := this.svcMap.Load(key)
					if ok {
						topic := fmt.Sprintf("/sys/%s/%s/thing/download", pKey, dName)
						pmsg := this.pm.PMPoolGet()
						pmsg.SetQoS(message.QosAtLeastOnce)
						pmsg.SetTopic([]byte(topic))
						pmsg.SetPayload(msg.Msg.Payload())
						err := svc.(*service).publish(pmsg, svc.(*service).onComplete)
						if err != nil {
							log.Errorf("biz.Push | productKey %s deviceName %s mid %s publish msg FAILED,err:%v", pKey, dName, err)
						}
						this.pm.PMPoolPut(pmsg)
						log.Debugf("publish message success, key: %s, topic: %s, path: %+v", key, topic, path)
					}else{
						log.Errorf("device is offline, key: %s, topic: %s, path: %+v", key, topic, path)
					}
				}
			}
		}()
	}
}