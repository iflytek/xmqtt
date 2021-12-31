package mongodbMngr

import (
	"git.iflytek.com/HY_XIoT/core/utils/log"
	"github.com/globalsign/mgo"
	"io"
	"time"
)

var sessionChanMap = make(map[*mgo.Session]chan int)

type Mongo struct {
	MgoSession  *mgo.Session
	MgoAddr     string
	Collection  *mgo.Collection
	SessionMode mgo.Mode
}

func newMongo(mgoAddr string, mgoSession *mgo.Session, collection *mgo.Collection, mode mgo.Mode) *Mongo {
	return &Mongo{
		MgoSession:  mgoSession,
		MgoAddr:     mgoAddr,
		Collection:  collection,
		SessionMode: mode,
	}
}

func MonitorSession(session *mgo.Session) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Warnf("mongodbMngr.MonitorSession | recover err:%s", err)
			}
		}()
		c := time.Tick(time.Second * 3)
		exit := make(chan int)
		sessionChanMap[session] = exit
		for {
			select {
			case <-c:
				if err := session.Ping(); err != nil && err == io.EOF {
					log.Warnf("mongodbMngr.MonitorSession | session Ping err:%s", err.Error())
					session.Refresh()
				} else {
					//log.Debug("monitorSession | session ok")
				}
			case <-exit:
				log.Warnln("mongodbMngr.MonitorSession | exit")
				break
			}
		}
	}()
}

func UnMonitorSession(session *mgo.Session) {
	log.Infoln("mongodbMngr.RemoveMonitorSession | enter")
	exit, ok := sessionChanMap[session]
	if ok {
		exit <- 1
	} else {
		log.Warnln("mongodbMngr.RemoveSession | not find session")
	}
}
