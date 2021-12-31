package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"xmqtt/utils/log"
)

type Configuration struct {
	KeepAlive         int
	ConnectTimeout    int
	TimeoutRetries    int
	MaxBufferSize     int
	SessionsProvider  string
	WsAddr            string // HTTPS websocket address eg. :8080
	WssAddr           string // HTTPS websocket address, eg. :8081
	RootCertPath      string // root cert
	TLsPublicCertPath string // server cert
	TLsPrivateKeyPath string // server key
	WssPublicCertPath string
	WssPrivateKeyPath string

	LogPath string

	NetCard         string
	PublicNetCard   string
	PublicNetworkIP string
	Prof bool

	AuthSvr   string
	UploadSvr string
	AckTimeout        int
	Interval int
	BucketNum int
}

const (
	// value of default configuration
	DefaultKeepAlive        = 300
	DefaultConnectTimeout   = 2
	DefaultAckTimeout       = 5
	DefaultTimeoutRetries   = 3
	DefaultMaxBufferSize    = 1024 * 256
	DefaultSessionsProvider = "mem"

	ROOT = "server"

	LogPath           = "logPath"
	LogLevel          = "logLevel"
	KeepAlive         = "keepAlive"
	ConnectTimeout    = "connectTimeout"
	AckTimeout        = "ackTimeout"
	TimeoutRetries    = "timeoutRetries"
	MaxBufferSize     = "maxBufferSize"
	SessionsProvider  = "sessionsProvider"
	CpuProfile        = "cpuProfile"
	MemProfile        = "memProfile"
	RootCertPath      = "rootCertPath"
	TlsPublicCertPath = "tlsPublicCertPath"
	TlsPrivateKeyPath = "tlsPrivateKeyPath"
	WssPublicCertPath = "wssPublicCertpath"
	WssPrivateKeyPath = "wssPrivateKeypath"
	NetCard           = "netcard"
	PublicNetCard     = "publicNetcard"
	PublicNetworkIP   = "publicNetworkIP"
)


func NewOptions() (*Configuration, error) {
	conf := &Configuration{}
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, errors.Wrap(err, "NewOptions | read config failed")
	}

	logPath := viper.GetString("log.path")
	logLevel := viper.GetString("log.level")
	log.NewLogger(logPath, logLevel)

	conf.KeepAlive = viper.GetInt("mqtt.keepalive")
	if conf.KeepAlive == 0 {
		conf.KeepAlive = DefaultKeepAlive
	}

	conf.ConnectTimeout = viper.GetInt("mqtt.connectTimeout")
	if conf.ConnectTimeout == 0 {
		conf.ConnectTimeout = DefaultConnectTimeout
	}

	conf.AckTimeout = viper.GetInt("mqtt.ackTimeout")
	if conf.AckTimeout == 0 {
		conf.AckTimeout = DefaultAckTimeout
	}

	conf.BucketNum = viper.GetInt("timeWheel.bucketNum")
	if conf.BucketNum == 0 {
		conf.BucketNum = 360
	}

	conf.Interval = viper.GetInt("timeWheel.interval")
	if conf.Interval == 0 {
		conf.Interval = 200
	}

	conf.TimeoutRetries = viper.GetInt("mqtt.retries")
	if conf.TimeoutRetries == 0 {
		conf.TimeoutRetries = DefaultTimeoutRetries
	}

	conf.TimeoutRetries = viper.GetInt("mqtt.retries")
	if conf.TimeoutRetries == 0 {
		conf.TimeoutRetries = DefaultTimeoutRetries
	}

	conf.MaxBufferSize = viper.GetInt("mqtt.maxBufferSize")
	if err == nil {
		conf.MaxBufferSize = DefaultMaxBufferSize
	}

	conf.SessionsProvider = DefaultSessionsProvider
	conf.SessionsProvider = viper.GetString("mqtt.sessionProvider")
	if conf.SessionsProvider == "" {
		conf.SessionsProvider = DefaultSessionsProvider
	}

	conf.RootCertPath = viper.GetString("cert.caPath")
	if conf.RootCertPath == "" {
		conf.RootCertPath = "/opt/cert/ca.crt"
	}

	conf.TLsPublicCertPath = viper.GetString("cert.serverCrtPath")
	if conf.TLsPublicCertPath == "" {
		conf.TLsPublicCertPath = "/opt/cert/server.crt"
	}

	conf.TLsPrivateKeyPath = viper.GetString("cert.serverKeyPath")
	if conf.TLsPrivateKeyPath == "" {
		conf.TLsPrivateKeyPath = "/opt/cert/server.key"
	}

	conf.WssPublicCertPath = viper.GetString("cert.wssCrtPath")
	if conf.WssPublicCertPath == "" {
		conf.WssPublicCertPath = "/opt/cert/wss.crt"
	}

	conf.WssPrivateKeyPath = viper.GetString("cert.wssKeyPath")
	if conf.WssPrivateKeyPath == "" {
		conf.WssPrivateKeyPath = "/opt/cert/wss.key"
	}

	conf.NetCard = viper.GetString("server.netCard")
	if err != nil {
		conf.NetCard = "eth0"
	}

	conf.PublicNetCard = viper.GetString("server.publicNetCard")
	if err != nil {
		conf.PublicNetCard = "eth0"
	}

	conf.Prof = viper.GetBool("server.Prof")
	log.Infof("NewOptions | Init Conf Success, conf:%+v", conf)
	return conf, nil
}
