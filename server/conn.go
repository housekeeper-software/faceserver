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
	pongWait   = 60 * time.Second    //pong接收的超时时间
	pingPeriod = (pongWait * 9) / 10 //ping发送周期
)

//wsConn 连接对象
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

//初始化一个新的连接
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

//关闭一个连接
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

//启动一个连接
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

		//启动读取协程
		go w.handleRead(done)

		for {
			select {
			case <-w.server.ctx.Done():
				//server will exit
				w.close()
				//此刻不能退出，需要等待读协程退出
			case <-done:
				//读取协程退出了，我们可以真正退出
				w.close()
				return
			case <-ticker.C:
				//发送ping包
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

	/*
		第一个包必须在指定时间内到达，否则就关闭连接
		这可以防止恶意连接
	*/
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
			glog.V(LERROR).Infof("json.Unmarshal %+v", err)
			return
		}
		msg.ConnId = w.seq
		//我们生成一个唯一的请求id
		msg.ReqId = GetIdInstance().Get()
		//提交给人脸特征提取模块处理
		face.GetFaceInstance().DoFeature(msg)
		glog.V(LVERBOSE).Infof("new request[%s] from:%s", msg.ID, w.addr)
	}
}

func (w *wsConn) send(resp face.Response) {
	w.writeCh <- resp
}
