package push

import (
	"errors"
	"strings"
	"sync/atomic"

	"git.iflytek.com/HY_XIoT/core/utils/log"
)

var ErrNoAvaliableServer = errors.New("no availiable server")
var ErrNoInit = errors.New("clientMgr no init")

type ClientMgr struct {
	clients []*client

	addrList []string
	inited   int32
	idx      int64 // 当前选择的client
	exitCH   chan struct{}
}

func NewClientMgr() *ClientMgr {
	return &ClientMgr{idx: 0, exitCH: make(chan struct{})}
}

func (cm *ClientMgr) Init(addrs string) {

	// 初始化一次
	if !atomic.CompareAndSwapInt32(&cm.inited, 0, 1) {
		return
	}
	// 根据","切割地址
	cm.addrList = strings.Split(addrs, ",")
	log.Warnf("ClientMgr.Init | addrs:%s", addrs)

	// 初始化客户端
	var cli *client
	for _, addr := range cm.addrList {
		cli = newClient(addr)
		cli.init()
		cm.clients = append(cm.clients, cli)
	}
}

func (cm *ClientMgr) Fini() {
	if !atomic.CompareAndSwapInt32(&cm.inited, 1, 0) {
		return
	}

	for _, cli := range cm.clients {
		cli.fini()
	}
}

func (cm *ClientMgr) Push(pid, did string, message []byte) error {
	cli := cm.getClient()
	if cli == nil {
		return ErrNoAvaliableServer
	}
	return cli.push(pid, did, message)
}

// 获取一个可用的client，有多个Client时轮训使用Client
func (cm *ClientMgr) getClient() *client {
	if len(cm.clients) == 0 {
		return nil
	}

	clientCount := int32(len(cm.clients))
	index := int32(atomic.AddInt64(&cm.idx, 1) % int64(clientCount))

	// client健康检查
	for i := int32(0); i < clientCount; i++ {
		client := cm.clients[(index+i)%clientCount]
		if client.state() {
			log.Debugf("getClient | get client:%s", client.svrAddr)
			return client
		}
	}
	return nil
}

var defaultClientMgr *ClientMgr

func Init(addrs string) error {
	if defaultClientMgr != nil {
		return nil
	}

	defaultClientMgr = NewClientMgr()
	defaultClientMgr.Init(addrs)
	return nil
}

func Fini() {
	defaultClientMgr.Fini()
}

func Push(pid, did string, message []byte) error {
	return defaultClientMgr.Push(pid, did, message)
}
