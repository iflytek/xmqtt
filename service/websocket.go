package service

import (
	"io"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/websocket"
	"xmqtt/utils/log"
)

func (this *Server) addWebsocketHandler(urlPattern string, uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		log.Errorf("surgemq/main: %v", err)
		return err
	}

	h := func(ws *websocket.Conn) {
		this.websocketTcpProxy(ws, u.Scheme, u.Host)
	}
	http.Handle(urlPattern, websocket.Handler(h))
	return nil
}

/* start a listener that proxies websocket <-> tcp */
func (this *Server) listenAndServeWebsocket(addr string) error {
	return http.ListenAndServe(addr, nil)
}

/* starts an HTTPS listener */
func (this *Server) listenAndServeWebsocketSecure(addr string, cert string, key string) error {
	return http.ListenAndServeTLS(addr, cert, key, nil)
}

/* copy from websocket to writer, this copies the binary frames as is */
func (this *Server) ioCopyWS(src *websocket.Conn, dst io.Writer) (int, error) {
	var buffer []byte
	count := 0
	for {
		err := websocket.Message.Receive(src, &buffer)
		if err != nil {
			return count, err
		}
		n := len(buffer)
		count += n
		i, err := dst.Write(buffer)
		if err != nil || i < 1 {
			return count, err
		}
	}
	return count, nil
}

/* copy from reader to websocket, this copies the binary frames as is */
func (this *Server) ioWSCopy(src io.Reader, dst *websocket.Conn) (int, error) {
	buffer := make([]byte, 2048)
	count := 0
	for {
		n, err := src.Read(buffer)
		if err != nil || n < 1 {
			return count, err
		}
		count += n
		err = websocket.Message.Send(dst, buffer[0:n])
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

/* handler that proxies websocket <-> unix domain socket */
func (this *Server) websocketTcpProxy(ws *websocket.Conn, nettype string, host string) error {
	client, err := net.Dial(nettype, host)
	if err != nil {
		return err
	}
	defer client.Close()
	defer ws.Close()
	chDone := make(chan bool)

	go func() {
		this.ioWSCopy(client, ws)
		chDone <- true
	}()
	go func() {
		this.ioCopyWS(ws, client)
		chDone <- true
	}()
	<-chDone
	return nil
}
