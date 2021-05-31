// +build windows

package shell

import (
	"github.com/Microsoft/go-winio"
	"net"
)

var shellName = `\\.\pipe\%s.sock`

func createServer(name string, args ...interface{}) (net.Listener, error) {
	conf := &winio.PipeConfig{
		MessageMode: false,
	}
	for _, v := range args {
		if s, ok := v.(string); ok {
			conf.SecurityDescriptor = s
		} else if s, ok := v.(bool); ok {
			conf.MessageMode = s
		} else if s, ok := v.(int); ok {
			conf.InputBufferSize = int32(s)
			conf.OutputBufferSize = int32(s)
		} else if s, ok := v.(*winio.PipeConfig); ok {
			return winio.ListenPipe(name, s)
		}
	}
	return winio.ListenPipe(name, conf)
}

func createClient(name string) (net.Conn, error) {
	return winio.DialPipe(name, nil)
}
