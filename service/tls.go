package service

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"xmqtt/utils/log"
	"io/ioutil"
	"net"
	"net/url"
	"sync/atomic"
	"time"
)


func (this *Server) runTLS(addr string, cert string, key string) error {
	go this.listenAndServeTLS(addr, cert, key)

	return nil
}

func (this *Server) runTLSSecure(addr string, rootCrt, cert string, key string) error {
	go this.listenAndServeTLSSecure(addr, rootCrt, cert, key)

	return nil
}

// tls listen and serve
func (this *Server) listenAndServeTLS(addr string, cert, key string) error {
	defer atomic.CompareAndSwapInt32(&this.tlsrunning, 1, 0)
	if !atomic.CompareAndSwapInt32(&this.tlsrunning, 0, 1) {
		return fmt.Errorf("tls.ListenAndServeTLS: TLS Server is already running")
	}

	// 加载服务端证书
	log.Debugf("tls.ListenAndServeTLS | addr:%s publicCert:%s privateKey:%s", addr, cert, key)
	crt, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		log.Errorf("tls.ListenAndServeTLS | tls.LoadX509KeyPair err:%s", err.Error())
		return err
	}
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{crt}
	// Time returns the current time as the number of seconds since the epoch.
	// If Time is nil, TLS uses time.Now.
	tlsConfig.Time = time.Now
	// Rand provides the source of entropy for nonces and RSA blinding.
	// If Rand is nil, TLS uses the cryptographic random reader in package
	// crypto/rand.
	// The Reader must be safe for use by multiple goroutines.
	tlsConfig.Rand = rand.Reader

	u, err := url.Parse(addr)
	if err != nil {
		log.Errorf("tls.ListenAndServeTLS | url.Parse failed:%s", err.Error())
		return err
	}

	this.lnTls, err = tls.Listen(u.Scheme, u.Host, tlsConfig)
	if err != nil {
		log.Errorf("tls.ListenAndServeTLS | tls.Listen err:%s", err.Error())
		return err
	}
	defer this.lnTls.Close()

	for {
		conn, err := this.lnTls.Accept()
		if err != nil {
			continue
		} else {
			go this.handleConnection(conn)
		}
	}
}

// secure tls listen and serve
func (this *Server) listenAndServeTLSSecure(addr string, rootCrt, cert, key string) error {
	defer atomic.CompareAndSwapInt32(&this.tlssrunning, 1, 0)
	if !atomic.CompareAndSwapInt32(&this.tlssrunning, 0, 1) {
		return fmt.Errorf("tls.ListenAndServeTLSSecure: Secure TLS Server is already running")
	}

	log.Debugf("tls.ListenAndServeTLSSecure | addr:%s rootCrt:%s publicCert:%s privateKey:%s", addr, rootCrt, cert, key)
	buf, err := ioutil.ReadFile(rootCrt) // 读取根证书
	if err != nil {
		log.Errorf("tls.ListenAndServeTLSSecure | ioutil.ReadFile[%s] err:%v", rootCrt, err)
		return err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(buf)

	// 加载服务端证书
	crt, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		log.Errorf("tls.ListenAndServeTLSSecure | tls.LoadX509KeyPair err:%s", err.Error())
		return err
	}
	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = []tls.Certificate{crt}
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	tlsConfig.ClientCAs = pool
	// Time returns the current time as the number of seconds since the epoch.
	// If Time is nil, TLS uses time.Now.
	tlsConfig.Time = time.Now
	// Rand provides the source of entropy for nonces and RSA blinding.
	// If Rand is nil, TLS uses the cryptographic random reader in package
	// crypto/rand.
	// The Reader must be safe for use by multiple goroutines.
	tlsConfig.Rand = rand.Reader

	u, err := url.Parse(addr)
	if err != nil {
		log.Errorf("tls.ListenAndServeTLSSecure | url.Parse failed:%s", err.Error())
		return err
	}

	this.lnTls, err = tls.Listen(u.Scheme, u.Host, tlsConfig)
	if err != nil {
		log.Errorf("tls.ListenAndServeTLSSecure | tls.Listen err:%s", err.Error())
		return err
	}
	defer this.lnTls.Close()

	for {
		conn, err := this.lnTls.Accept()
		if err != nil {
			continue
		} else {
			go this.handleConnection(conn)
		}
	}
}

func (this *Server) handleClienttlsConnect(conn net.Conn) {
	defer conn.Close()
	buffer := make([]byte, 2048)
	for {
		len, err := conn.Read(buffer)
		if err != nil {
			break
		}
		//发送给客户端
		_, err = conn.Write([]byte(string(buffer[:len])))
		if err != nil {
			break
		}
	}
}
