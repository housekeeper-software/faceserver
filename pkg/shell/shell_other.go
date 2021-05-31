// +build !windows

package shell

import (
	"github.com/pkg/errors"
	"net"
)

var shellName = `@/tmp/%s.sock`

func createServer(name string, args ...interface{}) (net.Listener, error) {
	addr, err := net.ResolveUnixAddr("unix", name)
	if err != nil {
		return nil, errors.Wrap(err, "ResolveUnixAddr error\n")
	}
	return net.ListenUnix("unix", addr)
}

func createClient(name string) (net.Conn, error) {
	addr, err := net.ResolveUnixAddr("unix", name)
	if err != nil {
		return nil, errors.Wrap(err, "ResolveUnixAddr failed\n");
	}

	return net.DialUnix("unix", nil, addr)
}
