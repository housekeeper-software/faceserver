package server

import (
	"context"
	"faceserver/face"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"net/http"
	"sync"
	"time"
)

//server websocket服务器
type server struct {
	upgrader websocket.Upgrader
	server   http.Server
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	quit     chan struct{}
	addr     string
	conns    map[uint32]*wsConn //所有的连接
	seq      uint32
	mu       sync.Mutex
}

//创建服务器对象
func newServer(addr string) *server {
	s := &server{
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 30 * time.Second,
			ReadBufferSize:   4096,
			WriteBufferSize:  4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		ctx:    nil,
		cancel: nil,
		quit:   make(chan struct{}),
		addr:   addr,
		seq:    0,
		conns:  make(map[uint32]*wsConn),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

//启动服务器
func (s *server) start() {
	go func() {
		s.server = http.Server{Addr: s.addr}
		http.HandleFunc("/", s.handleConn)
		err := s.server.ListenAndServe()
		if err != nil {
			fmt.Printf("websocket listen[%s] err: %+v\n", s.addr, err)
		}
		s.quit <- struct{}{}
	}()
}

//结束服务器
func (s *server) close() error {
	err := s.server.Close()
	if err != nil {
		return errors.Wrap(err, "close websocket server failed\n")
	}
	<-s.quit
	fmt.Println("websocket:get server quit signal")
	s.cancel()
	s.wg.Wait()
	return nil
}

//处理新的连接
func (s *server) handleConn(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		glog.V(1).Infof("Upgrade failed : %+v\n", err)
		return
	}
	conn := newConn(ws, s.seq, s)
	conn.onClosed = s.onClosed
	s.onOpen(s.seq, conn)
	s.wg.Add(1)
	conn.start()
	s.seq++
}

func (s *server) onOpen(seq uint32, conn *wsConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[seq] = conn
}

func (s *server) onClosed(seq uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, seq)
}

//从connId找到对应的连接，并将response发送过去
func (s *server) send(connId uint32, resp face.Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.conns[connId]; ok {
		c.writeCh <- resp
	}
}

func (s *server) State() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.conns)
	return fmt.Sprintf("connection count: %d", n)
}
