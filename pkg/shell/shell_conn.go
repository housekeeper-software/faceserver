package shell

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
)

type connection struct {
	conn      net.Conn
	cid       uint32
	writeCh   chan string
	ctx       context.Context
	isClosed  bool
	onClosed  func(connId uint32)
	onMessage func(connId uint32, message string)
}

func newConnection(conn net.Conn, cid uint32, ctx context.Context) *connection {
	return &connection{
		conn:     conn,
		cid:      cid,
		ctx:      ctx,
		isClosed: false,
		writeCh:  make(chan string, 10),
	}
}

func (c *connection) close() {
	if c.isClosed {
		return
	}
	_ = c.conn.Close()
	c.isClosed = true
}

func (c *connection) start() {
	go func() {
		defer func() {
			if v := recover(); v != nil {
				log.Println("capture a panic in shell Client:", v)
			}
			c.close()
			c.onClosed(c.cid)
			fmt.Printf("shell client[%d] exit\n", c.cid)
		}()

		done := make(chan struct{})

		go handleRead(c, done)

		for {
			select {
			case <-c.ctx.Done():
				//server will quit
				c.close()
			case <-done:
				return
			case data := <-c.writeCh:
				buf, err := packet(data)
				if err == nil {
					_, err = c.conn.Write(buf)
				}
			}
		}
	}()
}

func handleRead(c *connection, done chan<- struct{}) {
	r := bufio.NewReader(c.conn)
	for {
		msg, err := unPacket(r)
		if err != nil {
			done <- struct{}{}
			return
		}
		c.onMessage(c.cid, msg)
	}
}
