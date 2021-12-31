package main

import (
	"encoding/json"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"xmqtt/utils/log"
	"time"
)

type Options struct {
	PKey string
	DName string
	TPkey string
	TDame string
	MqttAddr string
}

type MQTTClient struct {
	Opt *Options
	Client        MQTT.Client
	UploadTopic   string
	DownloadTopic string
}

func (m *MQTTClient) InitClient() error {
	password := "123456"
	userName := m.Opt.DName + "&" + m.Opt.PKey
	opts := MQTT.NewClientOptions()
	opts.AddBroker(m.Opt.MqttAddr)
	opts.SetClientID("demo")
	opts.SetUsername(userName)
	opts.SetPassword(password)
	m.Client = MQTT.NewClient(opts)

	if len(m.Opt.TPkey) > 0 && len(m.Opt.TDame) > 0 {
		m.UploadTopic = fmt.Sprintf("/sys/%s/%s/thing/upload/%s/%s", m.Opt.PKey, m.Opt.DName, m.Opt.TPkey, m.Opt.TDame)
	}else{
		m.UploadTopic = fmt.Sprintf("/sys/%s/%s/thing/upload", m.Opt.PKey, m.Opt.DName)
	}

	m.DownloadTopic = fmt.Sprintf("/sys/%s/%s/thing/download", m.Opt.PKey, m.Opt.DName)
	return nil
}

func (m *MQTTClient) Init() error {
	return m.InitClient()
}

func (m *MQTTClient) Connect() error {
	if token := m.Client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTTClient) Sub(handle MQTT.MessageHandler) error {
	if token := m.Client.Subscribe(m.DownloadTopic, 1, handle); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTTClient) Pub(payload []byte) error {
	log.Debugf("mqtt client pub message: %s", payload)
	if token := m.Client.Publish(m.UploadTopic, 1, true, payload); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

type Message struct {
	Content string
}

type SourceClient struct {
	MqttCli *MQTTClient
	Msg Message
}

func (p *SourceClient) MessageHandler (client MQTT.Client, message MQTT.Message) {
	log.Debugf("receive message: %s,topic: %s", message.Payload(), message.Topic())

}

func (p *SourceClient) BuildMsg() ([]byte, error) {
	msgByte, err := json.Marshal(p.Msg)
	if err != nil{
		log.Errorf("SourceClient | BuildMsg failed, err: %s", err.Error())
		return nil, err
	}
	return msgByte, nil
}

func (p *SourceClient) SendMessage() error {
	err := p.MqttCli.Connect()
	if err != nil {
		return errors.Wrap(err, "connect failed")
	}

	err = p.MqttCli.Sub(p.MessageHandler)
	if err != nil {
		return errors.Wrap(err, "sub failed")
	}

	payload, err := p.BuildMsg()
	if err != nil {
		return errors.Wrap(err, "BuildMsg failed")
	}

	ticker := time.NewTicker(time.Duration(200) * time.Millisecond)
	for  {
		select {
		case <-ticker.C:
			err = p.MqttCli.Pub(payload)
			if err != nil {
				return errors.Wrapf(err, "pub failed, payload: %s", payload)
			}
		}
	}
	return nil
}

type TargetClient struct {
	MqttCli *MQTTClient
	Opt     *Options
}

func (p *TargetClient) MessageHandler (client MQTT.Client, message MQTT.Message) {
	log.Debugf("receive message: %s,topic: %s", message.Payload(), message.Topic())
}

func (p *TargetClient) ConnectAndSub() error {
	err := p.MqttCli.Connect()
	if err != nil {
		return errors.Wrap(err, "connect failed")
	}

	err = p.MqttCli.Sub(p.MessageHandler)
	if err != nil {
		return errors.Wrap(err, "sub failed")
	}
	return nil
}

func main()  {
	log.NewLoggerInConsole("debug")
	optTarget := &Options{
		PKey:     "car",
		DName:    "test2",
		MqttAddr: "127.0.0.1:1883",
	}

	st := TargetClient{
		MqttCli: &MQTTClient{
			Opt:   optTarget,
		},
	}

	err := st.MqttCli.Init()
	if err != nil{
		log.Errorf("Init failed, err: %s", err.Error())
		return
	}

	err = st.ConnectAndSub()
	if err != nil{
		log.Errorf("send msg failed, err: %s", err.Error())
		return
	}

	opt := &Options{
		PKey:     "car",
		DName:    "test1",
		TPkey:    "car",
		TDame:    "test2",
		MqttAddr: "127.0.0.1:1883",
	}

	sc := SourceClient{
		MqttCli: &MQTTClient{
			Opt:   opt,
		},
		Msg:     Message{
			Content: "hello",
		},
	}

	err = sc.MqttCli.Init()
	if err != nil{
		log.Errorf("Init failed, err: %s", err.Error())
		return
	}

	err = sc.SendMessage()
	if err != nil{
		log.Errorf("send msg failed, err: %s", err.Error())
		return
	}

	select {}
}
