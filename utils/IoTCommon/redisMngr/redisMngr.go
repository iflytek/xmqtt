package redisMngr

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"git.iflytek.com/HY_XIoT/core/utils/log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gomodule/redigo/redis"
)

var casLuaScript = `
local val = redis.call('GET',KEYS[1])
if val == ARGV[1] then
local err = redis.call('SET',KEYS[1], ARGV[2])
	return 1
end
return 0`
var casLuaScriptSha1 string

var setLatestValueScript = `
local val=tostring(redis.call('get', KEYS[1]))
local delimiter=":"
if val == "false" then
  local rst=redis.call('set', KEYS[1], ARGV[2])
  if table.getn(rst) == 0 then
    return 0
  end
  return 1
else
    local result = {};
    for match in (val..delimiter):gmatch("(.-)"..delimiter) do
        table.insert(result, match);
    end
    local length=table.getn(result)
    local timestamp=tonumber(result[length])
    if timestamp == nil then
      return 1
    end
    local timestampNow=tonumber(ARGV[1])
    if timestampNow == nil then
      return 1
    end
    if timestamp < timestampNow then
      local rst=redis.call('set', KEYS[1], ARGV[2])
      if table.getn(rst) == 0 then
        return 0
      end
      return 1
    end
    return 1
end
`
var setLatestValueScriptSha1 string

//需要集成seelog

//Redis ...
type Redis struct {
	FailedCount  int64
	Retrytime    int64
	Pool         *redis.Pool
	addr         string
	password     string
	maxIdle      int
	maxActive    int
	idleTimeout  time.Duration
	mutexupdate  sync.Mutex
	contimeout   time.Duration
	readtimeout  time.Duration
	writetimeout time.Duration
}

func (r *Redis) initRedisPool() {
	r.Pool = &redis.Pool{
		MaxIdle:     r.maxIdle,
		MaxActive:   r.maxActive,
		IdleTimeout: r.idleTimeout,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", r.addr,
				redis.DialConnectTimeout(time.Millisecond*r.contimeout),
				redis.DialReadTimeout(time.Millisecond*r.readtimeout),
				redis.DialWriteTimeout(time.Millisecond*r.writetimeout))
			if err != nil {
				//	log.Warnf("redis Dial err:", err)
				return nil, err

			}

			if _, err = c.Do("AUTH", r.password); err != nil {
				c.Close()
				return nil, err
			}

			log.Infof("redis initRedisPool ok, addr:%s", r.addr)
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
		Wait: true,
	}
}

// GetConn 从redis pool中获取一个redis Conn
func (r *Redis) getConn() redis.Conn {
	if r.Pool == nil {
		return nil
	}
	return r.Pool.Get()
}

// Do ...
func (r *Redis) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	conn := r.getConn()
	if conn == nil {
		r.UpdateFailed()
		return nil, fmt.Errorf("redis.Pool GetConn failed")
	}
	defer conn.Close()
	reply, err = conn.Do(commandName, args...)
	if err != nil {
		r.UpdateFailed()
	}
	return
}

// Send ...
func (r *Redis) Send(commandName string, args ...interface{}) error {
	conn := r.getConn()
	if conn == nil {
		r.UpdateFailed()
		return fmt.Errorf("redis.Pool GetConn failed")
	}
	defer conn.Close()
	err := conn.Send(commandName, args...)
	if err != nil {
		r.UpdateFailed()
	}
	return nil
}

// Flush ...
func (r *Redis) Flush() error {
	conn := r.getConn()
	if conn == nil {
		r.UpdateFailed()
		return fmt.Errorf("redis.Pool GetConn failed")
	}
	defer conn.Close()
	err := conn.Flush()
	if err != nil {
		r.UpdateFailed()
	}
	return nil
}

// Receive ...
func (r *Redis) Receive() (reply interface{}, err error) {
	conn := r.getConn()
	if conn == nil {
		r.UpdateFailed()
		return nil, fmt.Errorf("redis.Pool GetConn failed")
	}
	defer conn.Close()
	reply, err = conn.Receive()
	if err != nil {
		r.UpdateFailed()
	}
	return
}

func (r *Redis) setCAS(key string, old, new interface{}) (reply interface{}, err error) {
	if casLuaScriptSha1 == "" {
		casLuaScriptSha1 = generateSha1(casLuaScript)
	}
	reply, err = r.Do("EVALSHA", casLuaScriptSha1, 1, key, old, new)
	if err != nil {
		//	log.Debugf("EVALSHA err:%s", err.Error())
		return r.Do("EVAL", casLuaScript, 1, key, old, new)
	}
	return
}

func (r *Redis) setLatestValue(key string, timestamp, value interface{}) (reply interface{}, err error) {
	if setLatestValueScriptSha1 == "" {
		setLatestValueScriptSha1 = generateSha1(setLatestValueScript)
	}
	reply, err = r.Do("EVALSHA", setLatestValueScriptSha1, 1, key, timestamp, value)
	if err != nil {
		//	log.Debugf("EVALSHA err:%s", err.Error())
		return r.Do("EVAL", setLatestValueScript, 1, key, timestamp, value)
	}
	return
}

func (r *Redis) UpdateOk() {
	atomic.StoreInt64(&(r.FailedCount), 0)
	//	log.Critical("redis connect update failed")
}

func (r *Redis) UpdateFailed() {
	//fmt.Println("update failed")
	//	log.Errorf("redis connect update failed ,times %d", r.FailedCount)
	atomic.AddInt64(&(r.FailedCount), 1)
	//atomic.StoreInt64(&(r.Retrytime), time.Now().Unix() + 30)
}

// Close ...
func (r *Redis) Close() {
	if r.Pool != nil {
		r.Pool.Close()
		r.Pool = nil
	}
}

// NewRedis ...
func NewRedis(addr, password string, maxIdle, maxActive int, idleTimeout, contimeout, readtimeout,
	writetimeout time.Duration) *Redis {
	r := &Redis{addr: addr, password: password, maxIdle: maxIdle, maxActive: maxActive, idleTimeout: idleTimeout,
		contimeout: contimeout, readtimeout: readtimeout, writetimeout: writetimeout,
		FailedCount: 0, Retrytime: 0}
	r.initRedisPool()
	return r
}

func generateSha1(input string) (sha string) {
	hasher := sha1.New()
	hasher.Write([]byte(input))
	sha = hex.EncodeToString(hasher.Sum(nil))
	return
}

//注意maxfailedcount 需要8字节对齐，否则无法使用atomic
type RedisMngr struct {
	maxfailedcount int64
	redisnode      []*Redis
	addrlen        int
	nodeindex      int64
	retrychan      chan *Redis
	stop           bool
	lockadd        sync.Mutex
	retrygroup     sync.WaitGroup
	retrytime      int64
	retryLock 	   sync.Mutex
	retryNode 	   map[string]bool
}

func (rm *RedisMngr) Init(addr, password string, maxIdle, maxActive int, idleTimeout, contimeout, readtimeout,
	writetimeout time.Duration) error {
	rm.maxfailedcount = 5
	rm.nodeindex = 0
	rm.retrytime = 10
	rm.retryNode = make(map[string]bool)
	if contimeout == 0 {
		contimeout = 100
	}
	if readtimeout == 0 {
		readtimeout = 200
	}
	if writetimeout == 0 {
		writetimeout = 200
	}
	var err error
	if addr == "" {
		return errors.New("redis addr is empty")
	}

	s := strings.Split(addr, ",")
	rm.addrlen = len(s)
	rm.redisnode = make([]*Redis, rm.addrlen)

	for i := 0; i < rm.addrlen; i++ {
		if s[i] == "" {
			return errors.New("addr is empty")
		}
		rm.redisnode[i] = NewRedis(s[i], password, maxIdle, maxActive, idleTimeout, contimeout, readtimeout, writetimeout)
	}
	rm.retrychan = make(chan *Redis, rm.addrlen)

	// 线程循环尝试down掉的节点
	go func() {
		rm.retrygroup.Add(1)
		defer rm.retrygroup.Done()
		retrymap := make(map[string]*Redis)
		timewait := time.NewTicker(time.Duration(rm.retrytime) * time.Second)
		for !rm.stop {
			select {
			case ch := <-rm.retrychan:
				retrymap[ch.addr] = ch
			case <-timewait.C:
				for k, v := range retrymap {
					v.Close()
					v.initRedisPool()
					_, err := v.Do("GET", "test")
					if err != nil {
						continue
					}
					v.UpdateOk()
					delete(retrymap, k)

					log.Infof("delete redis node: %s from %+v", v.addr, rm.retryNode)
					delete(rm.retryNode, v.addr)

					//线上twemproxy不支持PING操作
					/*
						if rep == "PONG" {

						}
					*/
				}
			}
		}
	}()
	return err
}

func (rm *RedisMngr) SetLatestValue(key string, timestamp, value interface{}) (reply interface{}, err error) {
	redisNode := rm.GetServerNode()

	for i := 0; i < rm.addrlen; i++ {
		if redisNode == nil {
			return "", errors.New("no redis server vaild")
		}
		reply, err = redisNode.setLatestValue(key, timestamp, value)
		if err == nil {
			break
		}
		//有重复获取错误节点的风险，在错误节点剔除之前，有待后续优化
		redisNode = rm.GetServerNode()
	}

	return
}

func (rm *RedisMngr) SetCAS(key string, old, new interface{}) (reply interface{}, err error) {
	redisNode := rm.GetServerNode()

	for i := 0; i < rm.addrlen; i++ {
		if redisNode == nil {
			return "", errors.New("no redis server vaild")
		}
		reply, err = redisNode.setCAS(key, old, new)
		if err == nil {
			break
		}
		//有重复获取错误节点的风险，在错误节点剔除之前，有待后续优化
		redisNode = rm.GetServerNode()
	}
	return
}

func (rm *RedisMngr) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	//index := atomic.AddInt64(&(rm.nodeindex), 0)
	//Fixme fuck
	redisNode := rm.GetServerNode()
	for i := 0; i < rm.addrlen; i++ {
		if redisNode == nil {
			return "", errors.New("no redis server vaild")
		}
		reply, err = redisNode.Do(commandName, args...)
		if err != nil {
			//有重复获取错误节点的风险，在错误节点剔除之前
			redisNode = rm.GetServerNode()
		} else {
			return
		}
	}
	return
}

func (rm *RedisMngr) Send(commandName string, args ...interface{}) error {
	redisNode := rm.GetServerNode()
	for i := 0; i < rm.addrlen; i++ {
		if redisNode == nil {
			return errors.New("no redis server vaild")
		}
		err := redisNode.Send(commandName, args...)
		if err != nil {
			//有重复获取错误节点的风险，在错误节点剔除之前
			redisNode = rm.GetServerNode()
		} else {
			return nil
		}
	}
	return nil
}

func (rm *RedisMngr) Flush() error {
	redisNode := rm.GetServerNode()
	for i := 0; i < rm.addrlen; i++ {
		if redisNode == nil {
			return errors.New("no redis server valid")
		}
		err := redisNode.Flush()
		if err != nil {
			//有重复获取错误节点的风险，在错误节点剔除之前
			redisNode = rm.GetServerNode()
		} else {
			return nil
		}
	}
	return nil
}

func (rm *RedisMngr) Receive() (reply interface{}, err error) {
	redisNode := rm.GetServerNode()
	for i := 0; i < rm.addrlen; i++ {
		if redisNode == nil {
			return "", errors.New("no redis server vaild")
		}
		reply, err = redisNode.Receive()
		if err != nil {
			//有重复获取错误节点的风险，在错误节点剔除之前
			redisNode = rm.GetServerNode()
		} else {
			return
		}
	}
	return
}

func (rm *RedisMngr) GetServerNode() (redisNode *Redis) {

	rm.lockadd.Lock()
	rm.nodeindex++
	index := int(rm.nodeindex) //int(atomic.AddInt64(&(rm.nodeindex), 1))
	if index >= rm.addrlen {
		//atomic.StoreInt64(&(rm.nodeindex), 0)
		rm.nodeindex = 0
	}
	rm.lockadd.Unlock()

	index = index % rm.addrlen
	idxindex := index
	for {
		redisNode = rm.redisnode[index]
		failedcount := atomic.LoadInt64(&(redisNode.FailedCount))

		if failedcount >= rm.maxfailedcount {
			//if failedcount == rm.maxfailedcount {
			rm.retryLock.Lock()
			if !rm.retryNode[redisNode.addr] {
				rm.retrychan <- redisNode
				rm.retryNode[redisNode.addr] = true
			}
			rm.retryLock.Unlock()
			//}
			index++
			index = index % rm.addrlen
			if idxindex == index {
				return nil
			}
			continue
		}
		break
	}
	return
}

func (rm *RedisMngr) Fini() {
	rm.stop = true
	rm.retrygroup.Wait()
	for _, v := range rm.redisnode {
		v.Close()
	}
}

var redisMngr *RedisMngr

func Init(addrs, password string, maxIdle, maxActive int, idleTimeout, contimeout, readtimeout,
	writetimeout time.Duration) error {
	redisMngr = &RedisMngr{}
	return redisMngr.Init(addrs, password, maxIdle, maxActive, idleTimeout, contimeout, readtimeout, writetimeout)
}

func Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	return redisMngr.Do(commandName, args...)
}

func Fini() {
	redisMngr.Fini()
}
