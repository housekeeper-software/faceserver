package shell

import (
	"bufio"
	"github.com/pkg/errors"
	"net"
	"time"
)

type client struct {
	conn    net.Conn
	timeout int //read timeout(second)
}

func NewClient() Client {
	return &client{
		conn:    nil,
		timeout: 0,
	}
}

func (c *client) Open(name string, timeout int) error {
	conn, err := createClient(name)
	if err != nil {
		return errors.Wrap(err, "createClient error\n")
	}
	c.conn = conn
	c.timeout = timeout
	return nil
}

func (c *client) Close() error {
	return c.conn.Close()
}

func (c *client) Write(message string) (string, error) {
	buf, err := packet(message)
	if err != nil {
		return "", errors.Wrap(err, "packet failed\n")
	}
	_, err = c.conn.Write(buf)
	if err != nil {
		return "", errors.Wrap(err, "write ipc failed\n")
	}
	if c.timeout > 0 {
		err = c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.timeout) * time.Second))
		if err != nil {
			return "", errors.Wrap(err, "SetReadDeadline failed\n")
		}
	}
	r := bufio.NewReader(c.conn)
	msg, err := unPacket(r)
	if err != nil {
		return msg, errors.Wrap(err, "unpacket failed\n")
	}
	return msg, nil
}
