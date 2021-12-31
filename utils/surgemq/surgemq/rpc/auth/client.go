package rpc

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
	inited       int32
	svrAddr      string
	authClient   pb.AuthClient
	conn         *grpc.ClientConn
	authClientOK bool
	exitCH       chan struct{}
}

func newClient(addr string) *client {
	return &client{authClientOK: false, svrAddr: addr, exitCH: make(chan struct{})}
}

func (c *client) init() {
	// 初始化一次
	if !atomic.CompareAndSwapInt32(&c.inited, 0, 1) {
		return
	}

	// 起一个goroutine负责连接管理
	go c.authClientWatcher()
}

// 负责authClient创建rpc连接，制止成功或者受到信号退出
func (c *client) authClientWatcher() {
	log.Debugf("client.authClientWatcher | enter")
	ticker := time.NewTicker(5 * time.Second)
	for {
		if !c.authClientOK {
			conn, err := grpc.Dial(c.svrAddr, grpc.WithInsecure())
			if err == nil {
				log.Debugf("client.authClientWatcher | grpc Dial %s success:", c.svrAddr)
				c.authClient = pb.NewAuthClient(conn)
				c.conn = conn
				c.authClientOK = true
				return
			}
			log.Errorf("client.authClientWatcher | Dial %s err:%v", c.svrAddr, err)
		}

		select {
		case <-c.exitCH:
			log.Warnf("client.authClientWatcher | exit...")
			ticker.Stop()
			return
		case <-ticker.C:
		}
	}
}

func (c *client) state() bool {
	if c.authClientOK {
		return c.conn.GetState() == connectivity.Ready
	}

	return false
}

// do rpc to c server
func (c *client) auth(clientId, productKey, deviceName, password string) error {
	log.Debugf("client | auth clientId:%s, productKey:%s, deviceName:%s, password:%s", clientId, productKey, deviceName, password)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &pb.AuthRequest{ClientID: clientId, ProductKey: productKey, DeviceName: deviceName, Password: password}

	_, err := c.authClient.Auth(ctx, req)
	if err != nil {
		log.Errorf("client | auth Client.DoAuth | err:%s", err.Error())
	}
	return err
}

func (c *client) login(productKey, deviceName, nodeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &pb.LoginRequest{NodeID: nodeID, ProductKey: productKey, DeviceName: deviceName}

	_, err := c.authClient.Login(ctx, req)
	if err != nil {
		log.Errorf("client | auth Client.DoLogin | err:%s", err.Error())
	}
	return err
}

func (c *client) logout(productKey, deviceName, nodeID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &pb.LogoutRequest{NodeID: nodeID, ProductKey: productKey, DevicenName: deviceName}

	_, err := c.authClient.Logout(ctx, req)
	if err != nil {
		log.Errorf("client | logout client.DoLogout | err:%s", err.Error())
	}
	return err
}

func (c *client) fini() {
	if !atomic.CompareAndSwapInt32(&c.inited, 1, 0) {
		return
	}
	close(c.exitCH)
}
