package main

import "C"
import (
	"faceserver/pkg/shell"
	"faceserver/server"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"log"
	"os"
)

type cmdLine struct {
	cmd     string
	listen  string
	version bool
}

var cmd cmdLine

func init() {
	flag.StringVar(&cmd.cmd, "cmd", "", "cmd")
	flag.StringVar(&cmd.listen, "listen", "", "listen")
	flag.BoolVar(&cmd.version, "version", false, "version")
}

func main() {
	flag.Parse()
	defer glog.Flush()

	if len(cmd.listen) > 0 {
		//表示是服务器侦听
		app := server.NewApp()
		err := app.Run(cmd.listen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%+v\n", err)
		}
		return
	}
	if cmd.version {
		//想要查询app版本
		fmt.Printf("\ntransit:\nversion=%s\nbuild time:%s\n", JXVERSION, JXBUILDTIME)
		return
	}
	//是shell进程
	if len(cmd.cmd) > 0 {
		client := shell.NewClient()
		err := client.Open(shell.MakeUniqueName(shell.Dir), 10)
		if err != nil {
			log.Fatal(err)
		}
		rv, err := client.Write(cmd.cmd)
		if err != nil {
			fmt.Printf("shell command exec failed: %+v\n", err)
		}
		fmt.Println(rv)
		client.Close()
	}
}
