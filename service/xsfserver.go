package service

import (
	"xmqtt/utils/log"
	"github.com/rfyiamcool/go-timewheel"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"xmqtt/config"
	"xmqtt/utils/xferror"
	"xmqtt/utils/IoTCommon/util"
)

type MqttXSFServer struct {
	Svr *Server
	conf *config.Configuration
}

var (
	Group        string
	Version      string
	CompanionUrl string

	hostIp string
	nodeID string // mqtt Node ID， 内网ip:port
)

type killed struct {
}

const (
	LocalDebuggingGroup        = "dev"
	LocalDebuggingVersion      = "1.0.9"
	LocalDebuggingCompanionUrl = "http://10.1.87.70:6868"
)

func (mxs *MqttXSFServer) Init()  {
	var err error
	mxs.conf, err = config.NewOptions()
	if err != nil {
		log.Fatalf("Init | NewOptions | init server config failed, err: %s", err.Error())
		os.Exit(1)
	}

	xerror.Init(xerror.XDomainxmqtt)
	svr := &Server{
		KeepAlive:        mxs.conf.KeepAlive,
		ConnectTimeout:   mxs.conf.ConnectTimeout,
		AckTimeout:       mxs.conf.AckTimeout,
		TimeoutRetries:   mxs.conf.TimeoutRetries,
		MaxBufferSize:    mxs.conf.MaxBufferSize,
		SessionsProvider: mxs.conf.SessionsProvider,
		mqttAddr: "tcp://:1883",
		tlsMqttAddr: "tcp://:8883",
	}

	MsgChan = make(chan UploadMsg, 1024)
	svr.pm.init()
	mxs.Svr = svr
	mxs.initTimeWheel()
	mxs.initNodeId()

	if mxs.conf.Prof {
		go func() {
			log.Infof("listen host ip: %s, port: 40000 for pprof", mxs.Svr.hostIP)
			http.ListenAndServe(hostIp+":40000", nil)
		}()
	}

	mxs.Svr.setNodeID(nodeID)
	if config.PublicNetworkIP != "" {
		mxs.Svr.PublicNetworkIP = config.PublicNetworkIP
	} else if config.PublicNetCard != "" {
		publicNetworkIP, err := util.GetIPFromNetCard(config.PublicNetCard)
		if err != nil {
			log.Errorf("Init | util.GetIPFromNetCard err:%v", err)
			os.Exit(1)
		}
		mxs.Svr.PublicNetworkIP = publicNetworkIP
	}


}

func (mxs *MqttXSFServer) initTimeWheel(){
	var err error
	svr := mxs.Svr
	svr.tw, err = timewheel.NewTimeWheel(time.Duration(mxs.conf.Interval)*time.Millisecond, mxs.conf.BucketNum)
	if err != nil {
		log.Fatalf("Init | NewTimeWheel failed, err:%s", err.Error())
		os.Exit(1)
	}
	svr.tw.Start()
}

func (mxs *MqttXSFServer) initNodeId(){
	var err error
	if config.NetCard != "" {
		hostIp, err = util.GetIPFromNetCard(config.NetCard)
		if err != nil {
			log.Errorf("Init | util.GetIPFromNetCard err:%v", err)
			os.Exit(1)
		}
	} else {
		mxs.Svr.hostIP = util.GetHostIP()
	}
	mxs.Svr.nodeID = hostIp + ":1884"
}

func (mxs *MqttXSFServer) Run() {
	go func() {
		mxs.Svr.Run(mxs.Svr.mqttAddr)
	}()

	if len(mxs.conf.WsAddr) > 0 || len(mxs.conf.WsAddr) > 0 {
		addr := "tcp://127.0.0.1:1883"
		mxs.Svr.addWebsocketHandler("/mqtt", addr)
		/* start a plain websocket listener */
		if len(mxs.conf.WsAddr) > 0 {
			go mxs.Svr.listenAndServeWebsocket(mxs.conf.WsAddr)
		}
		/* start a secure websocket listener */
		if len(mxs.conf.WsAddr) > 0 && len(mxs.conf.WssPublicCertPath) > 0 && len(mxs.conf.WssPrivateKeyPath) > 0 {
			go mxs.Svr.listenAndServeWebsocketSecure(mxs.conf.WsAddr, mxs.conf.WssPublicCertPath, mxs.conf.WssPrivateKeyPath)
		}
	}

	/* start a secure tls listener */
	if len(mxs.Svr.tlsMqttAddr) > 0 && len(mxs.conf.RootCertPath) > 0 && len(mxs.conf.WssPublicCertPath) > 0 && len(mxs.conf.WssPrivateKeyPath) > 0 {
		go func() {
			err := mxs.Svr.runTLSSecure(mxs.Svr.tlsMqttAddr, mxs.conf.RootCertPath, mxs.conf.WssPublicCertPath, mxs.conf.WssPrivateKeyPath)
			if err != nil {
				log.Errorf("xsf server.Run | svr.runTLSSecure err:%v", err)
			}
		}()
	}

	mxs.Svr.PushMsg()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("xmqtt quit")
		mxs.Svr.Close()
		os.Exit(0)
	}()
}
