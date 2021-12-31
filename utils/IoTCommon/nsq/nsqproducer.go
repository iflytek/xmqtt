package nsq

import (
	"errors"
	"fmt"
	"git.iflytek.com/HY_XIoT/core/utils/xferror"
	"sync/atomic"
	"time"

	"git.iflytek.com/HY_XIoT/core/utils/log"
	"github.com/nsqio/go-nsq"
)

var (
	MaxDefaultQue = 1024 * 1000
	ProduceNum    = 10

	ProduceCount int64 = 0
	AsyncSuccess int64 = 0
	AsyncFailed  int64 = 0
)

type Producer struct {
	producers map[string]*nsq.Producer
	config    *nsq.Config
	call      chan *nsq.ProducerTransaction
	Done      chan bool
	topic     string
	addrs     []string
	counter   uint64
	c         chan []byte
}

func NewNsqProducer(addrs []string, topic string, ch chan []byte) (*Producer, error) {
	cfg := nsq.NewConfig()
	cfg.WriteTimeout = 10 * time.Second

	p := &Producer{
		config:    cfg,
		producers: make(map[string]*nsq.Producer),
		call:      make(chan *nsq.ProducerTransaction, MaxDefaultQue),
		Done:      make(chan bool),
		topic:     topic,
		addrs:     addrs,
		counter:   0,
		c:         ch,
	}

	for _, addr := range addrs {
		producer, err := nsq.NewProducer(addr, cfg)
		if err != nil {
			xErr := xerror.Newf(xerror.XCodeNSQProductErr, "NsqProducer::NewNsqProducer | new nsq producer error:%s",
				err.Error())
			log.Errorln(xErr.Error())
			return nil, err
		}

		err = producer.Ping()
		if err != nil {
			xErr := xerror.Newf(xerror.XCodeNSQProductErr, "NsqProducer::NewNsqProducer |  nsq producer ping error:%s",
				err.Error())
			log.Errorln(xErr.Error())
			return nil, err
		}
		p.producers[addr] = producer
	}
	return p, nil
}

func (p *Producer) Produce() {

	go p.CheckAsync()

	for i := 0; i < ProduceNum; i++ {
		go func() {
			for {
				select {
				case m := <-p.c:
					atomic.AddInt64(&ProduceCount, 1)
					fmt.Printf("topic:" + p.topic + " msg:" + string(m) + "\n")

					err := p.PublishAsync(p.topic, m, p.call, "a")
					if err != nil {
						xErr := xerror.Newf(xerror.XCodeNSQProductErr,
							"NsqProducer::Produce |  nsq producer publish async error:%s",
							err.Error())
						log.Errorln(xErr.Error())
						continue
					}
				case <-p.Done:
					log.Errorf("NsqProducer::Produce | getting out\n")
					return
				}
			}
		}()
	}
}

func (p *Producer) CheckAsync() {
	for {
		select {
		case call := <-p.call:
			if call.Error != nil {
				xErr := xerror.Newf(xerror.XCodeNSQProductErr, "NsqProducer::CheckAsync | check async error:%s",
					call.Error.Error())
				log.Errorln(xErr.Error())
				atomic.AddInt64(&AsyncFailed, 1)
			} else {
				atomic.AddInt64(&AsyncSuccess, 1)
			}
		}
	}
}

// Publish 消息发布
func (p *Producer) Publish(topic string, body []byte) error {
	pd := p.getProducer()
	if pd == nil {
		return errors.New("NsqProducer::Publish | getProducer return nil\n")
	}
	return pd.Publish(topic, body)
}

// PublishAsync 异步
func (p *Producer) PublishAsync(topic string, body []byte, doneChan chan *nsq.ProducerTransaction,
	args ...interface{}) error {
	pd := p.getProducer()
	if pd == nil {
		return errors.New("NsqProducer::PublishAsync | getProducer return nil\n")
	}
	return pd.PublishAsync(topic, body, doneChan, args)
}

func (p *Producer) getProducer() *nsq.Producer {
	if len(p.addrs) == 0 {
		return nil
	}
	counter := atomic.AddUint64(&p.counter, 1)
	idx := counter % uint64(len(p.addrs))
	addr := p.addrs[idx]
	pd := p.producers[addr]

	return pd
}

// Fini ...
func (p *Producer) Fini() {
	for _, pd := range p.producers {
		pd.Stop()
	}

	for i := 0; i < ProduceNum; i++ {
		p.Done <- true
	}
	close(p.Done)
}
