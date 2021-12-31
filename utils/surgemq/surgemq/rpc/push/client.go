package push

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"git.iflytek.com/HY_XIoT/core/utils/log"
	pb "git.iflytek.com/HY_XIoT/core/xfyun/surgemq/surgemq/proto/device"
	"google.golang.org/grpc/connectivity"

	"google.golang.org/grpc"
)

var ErrAddrEmpty error = errors.New("server addr is empty")

type client struct {
	inited         int32
	svrAddr        string
	deviceClient   pb.DeviceClient
	conn           *grpc.ClientConn
	deviceClientOK bool
	exitCH         chan struct{}
}

func newClient(addr string) *client {
	return &client{deviceClientOK: false, svrAddr: addr, exitCH: make(chan struct{})}
}

func (c *client) init() {
	// 初始化一次
	if !atomic.CompareAndSwapInt32(&c.inited, 0, 1) {
		return
	}

	// 起一个goroutine负责连接管理
	go c.clientWatcher()
}

// 负责deviceClient创建rpc连接，制止成功或者受到信号退出
func (c *client) clientWatcher() {
	log.Debugf("client.deviceClientWatcher | enter")
	ticker := time.NewTicker(5 * time.Second)
	for {
		if !c.deviceClientOK {
			conn, err := grpc.Dial(c.svrAddr, grpc.WithInsecure())
			if err == nil {
				log.Debugf("client.deviceClientWatcher | Dial %s success:", c.svrAddr)
				c.deviceClient = pb.NewDeviceClient(conn)
				c.conn = conn
				c.deviceClientOK = true
				return
			}
			log.Errorf("client.deviceClientWatcher | Dial %s err:%v", c.svrAddr, err)
		}

		select {
		case <-c.exitCH:
			log.Warnf("client.deviceClientWatcher | exit...")
			ticker.Stop()
			return
		case <-ticker.C:
		}
	}
}

func (c *client) state() bool {
	if c.deviceClientOK {
		if c.conn.GetState() == connectivity.Ready {
			return true
		}
	}

	return false
}

func (c *client) push(productID, deviceID string, message []byte) error {
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	//req := &pb.PushRequest{ProductID: productID, DeviceID: deviceID, Data: message}
	//
	//_, err := c.deviceClient.Push(ctx, req)
	//if err != nil {
	//	log.Errorf("client.Push | err:%s", err.Error())
	//}
	return nil
}

func (c *client) fini() {
	if !atomic.CompareAndSwapInt32(&c.inited, 1, 0) {
		return
	}
	close(c.exitCH)
}
