//Inter-process communication, server and client
//windows: microsoft winio
//linux: unix socket
package shell

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Option int

const (
	//ipc name use exe file name ,does not contain path
	Global Option = iota
	//ipc name use exe directory name
	Dir
)

//make a unique ipc name for server and client base platform
func MakeUniqueName(o Option) string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)

	if o == Dir {
		d := filepath.Dir(path)
		m := md5.New()
		m.Write([]byte(d))
		s := strings.ToUpper(hex.EncodeToString(m.Sum(nil)))
		return fmt.Sprintf(shellName, s)
	}
	return fmt.Sprintf(shellName, filepath.Base(path))
}

type Callback interface {
	OnMessage(cid uint32, message string)
}

//ipc server interface
type Server interface {
	//start server,if windows platform, you can setup winio.PipeConfig
	Open(name string, args ...interface{}) error

	//write to connection with cid
	Write(cid uint32, message string) error

	//stop and wait all goroutine quit
	Close() error
}

type Client interface {
	//timeout: max time(second) wait to read
	Open(name string, timeout int) error

	//write to server
	Write(message string) (string, error)

	Close() error
}
