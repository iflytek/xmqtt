package nsq

import (
	"fmt"
	"git.iflytek.com/HY_XIoT/core/utils/xferror"
	"sync/atomic"

	"git.iflytek.com/HY_XIoT/core/utils/log"
	"github.com/nsqio/go-nsq"
	"github.com/urfave/cli"
)

//var defaultNsqds = cli.StringSlice{"172.16.154.105:4150", "172.16.154.105:4250"}
var defaultLookupds = cli.StringSlice{"172.16.154.105:4161"}
var ConsumeCount int64 = 0

type dispatch interface {
	DoMsg([]byte) error
}

type Consumer struct {
	consumer *nsq.Consumer
	config   *nsq.Config
	Attributes

	Dis dispatch
}

type Attributes struct {
	Topic       string
	Channel     string
	Routinenum  int
	Maxinflight int
}

func (n *Consumer) HandleMessage(message *nsq.Message) error {
	atomic.AddInt64(&ConsumeCount, 1)

	if err := n.Dis.DoMsg(message.Body); err != nil {
		xErr := xerror.Newf(xerror.XCodeNSQConsumeErr, "NsqConsumer::HandleMessage | consume message error:%s",
			err.Error())
		log.Errorln(xErr.Error())
		return err
	}

	return nil
}

func (n *Consumer) LogFailedMessage(message *nsq.Message) {
	log.Errorf("NsqConsumer::LogFailedMessage | Failed Message: %s", string(message.Body))
	//TODO: handle failedmessage
}

func NewNsqConsumer(topic, channel string, routinenum, maxinflight int) (*Consumer, error) {

	attrs := Attributes{
		Topic:       topic,
		Channel:     channel,
		Routinenum:  routinenum,
		Maxinflight: maxinflight,
	}

	cfg := nsq.NewConfig()
	cfg.MaxInFlight = attrs.Maxinflight
	//TODO:set config
	consumer, err := nsq.NewConsumer(attrs.Topic, attrs.Channel, cfg)

	if err != nil {
		log.Errorf("NewNsqConsumer | consumer error: %s\n", err.Error())
		fmt.Printf("new consumer error: %s", err.Error())
		return nil, err
	}

	n := &Consumer{
		Attributes: attrs,
		consumer:   consumer,
		config:     cfg,
	}

	consumer.AddConcurrentHandlers(n, n.Attributes.Routinenum)
	return n, nil
}

func (n *Consumer) Consume(nsqds, lookupds []string, routineNum int) {

	//nsq
	var err error
	fmt.Println(nsqds)

	if len(lookupds) > 0 {
		err = n.consumer.ConnectToNSQLookupds(lookupds)
	} else if len(nsqds) > 0 {
		err = n.consumer.ConnectToNSQDs(nsqds)
	} else {
		lookupds = defaultLookupds
		//注意 此处并不检查连接 只要地址格式正确 err==nil
		err = n.consumer.ConnectToNSQLookupds(lookupds)
	}

	if err != nil {
		log.Errorf("NsqConsumer::Consume | connect to NSQD/NSQLookUp error: %s", err.Error())
		//TODO: add log
	}
	<-n.consumer.StopChan
}

func (n *Consumer) Fini() {
	n.consumer.Stop()
}
