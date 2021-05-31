package shell

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"sync"
)

//windows,linux will implement individual
type ctor interface {
	createServer(name string, args ...interface{}) (net.Listener, error)
	createClient(name string) (net.Conn, error)
}

type server struct {
	conns    map[uint32]*connection
	listener net.Listener
	seq      uint32
	mu       sync.Mutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	quit     chan struct{}
	cb       Callback
}

func NewServer(cb Callback) Server {
	return &server{
		conns:    make(map[uint32]*connection),
		listener: nil,
		seq:      0,
		cb:       cb,
		quit:     make(chan struct{}),
	}
}

func (s *server) Write(cid uint32, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, err := s.get(cid)
	if err != nil {
		return errors.Wrapf(err, "connection[%d] destroy already\n", cid)
	}
	conn.writeCh <- message
	return nil
}

func (s *server) Open(name string, args ...interface{}) error {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	ln, err := createServer(name, args)
	if err != nil {
		return errors.Wrap(err, "createServer failed\n")
	}
	s.listener = ln

	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				s.quit <- struct{}{}
				fmt.Println("ipc server quit!")
				return
			}
			c := newConnection(conn, s.seq, s.ctx)
			c.onClosed = s.onClosed
			c.onMessage = s.onMessage
			s.add(c)
			s.seq++
			c.start()
		}
	}()
	return nil
}

func (s *server) Close() error {
	if s.listener == nil {
		return nil
	}
	err := s.listener.Close()
	if err != nil {
		return errors.Wrap(err, "close ipc server failed\n")
	}
	//wait listen loop break
	<-s.quit
	fmt.Println("get ipc server quit signal")
	//alert all connection quit
	s.cancel()
	//wait for all connection quit complete
	s.wg.Wait()
	fmt.Println("all connections quit")
	return nil
}

func (s *server) add(conn *connection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[conn.cid] = conn
	s.wg.Add(1)
}

func (s *server) get(cid uint32) (*connection, error) {
	if c, ok := s.conns[cid]; ok {
		return c, nil
	}
	return nil, errors.Errorf("shell client[%d] not found\n", cid)
}

func (s *server) onClosed(cid uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, cid)
	s.wg.Done()
}

func (s *server) onMessage(cid uint32, message string) {
	s.cb.OnMessage(cid, message)
}
