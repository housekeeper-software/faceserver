package server

import (
	"encoding/json"
	"faceserver/face"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type wsConn struct {
	conn        *websocket.Conn
	server      *server
	isClosed    bool
	writeCh     chan face.Response
	messageType int
	addr        string
	seq         uint32
	onClosed    func(seq uint32)
}

func newConn(conn *websocket.Conn, seq uint32, server *server) *wsConn {
	s := &wsConn{
		conn:        conn,
		seq:         seq,
		server:      server,
		isClosed:    false,
		messageType: websocket.TextMessage,
		addr:        conn.RemoteAddr().String(),
		writeCh:     make(chan face.Response),
	}
	return s
}

func (w *wsConn) close() {
	if w.isClosed {
		return
	}
	err := w.conn.Close()
	if err != nil {
		glog.V(LERROR).Infof("ws conn[%s] close failed: %+v", w.addr, err)
	}
	w.onClosed(w.seq)
	w.isClosed = true
}

func (w *wsConn) start() {
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			if v := recover(); v != nil {
				log.Println("capture a panic in wsConn:", v)
			}
			ticker.Stop()
			w.server.wg.Done()
			glog.V(LVERBOSE).Infof("ws conn[%s] routine exit", w.addr)
		}()

		done := make(chan struct{})

		go w.handleRead(done)

		for {
			select {
			case <-w.server.ctx.Done():
				//server will exit
				w.close()
				//此刻不能退出，需要等待读协程退出
			case <-done:
				w.close()
				return
			case <-ticker.C:
				_ = w.conn.WriteMessage(websocket.PingMessage, []byte{})
				glog.V(LVERBOSE).Infof("ws conn[%s] send ping message", w.addr)
			case data := <-w.writeCh:
				buf, err := json.Marshal(data)
				if err == nil {
					err = w.conn.WriteMessage(w.messageType, buf)
					if err != nil {
						glog.V(LERROR).Infof("ws conn[%s] write message failed:%+v", w.addr, err)
					} else {
						glog.V(LVERBOSE).Infof("request[%s] peer addr:%s done!", data.ID, w.addr)
					}
				}
			}
		}
	}()
}

func (w *wsConn) handleRead(done chan<- struct{}) {
	defer func() {
		done <- struct{}{}
	}()

	w.conn.SetReadDeadline(time.Now().Add(pongWait))

	w.conn.SetPongHandler(func(string) error {
		w.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		mt, data, err := w.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				glog.V(LERROR).Infof("ws conn[%s] read error: %v", w.addr, err)
			}
			return
		}
		w.messageType = mt
		msg := &face.Request{}
		if err := json.Unmarshal(data, msg); err != nil {
			glog.V(LERROR).Infof("BinaryMessage must use ProbufMessage!")
			return
		}
		msg.ConnId = w.seq
		msg.ReqId = GetIdInstance().Get()
		face.GetFaceInstance().DoFeature(msg)
		glog.V(LVERBOSE).Infof("new request[%s] from:%s", msg.ID, w.addr)
	}
}

func (w *wsConn) send(resp face.Response) {
	w.writeCh <- resp
}
