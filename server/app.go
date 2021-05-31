package server

import (
	"context"
	"faceserver/face"
	"faceserver/pkg/shell"
	"fmt"
	"github.com/golang/glog"
	"runtime"
	"strings"
	"time"
)

//App  应用程序对象
type App struct {
	ws     *server
	cmd    shell.Server
	ctx    context.Context
	cancel context.CancelFunc
}

//创建应用程序实例
func NewApp() *App {
	app := &App{}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	return app
}

//运行，需要指定服务器侦听地址，比如:9999
func (app *App) Run(addr string) error {
	defer func() {
		if app.ws != nil {
			app.ws.close()
		}
		face.GetFaceInstance().UnInit()

		if app.cmd != nil {
			app.cmd.Close()
		}
	}()
	//设置人脸特征模块的回调
	face.GetFaceInstance().OnCompleted = app.onCompleted
	err := face.GetFaceInstance().Init()
	if err != nil {
		return err
	}

	//启动shell
	app.cmd = shell.NewServer(app)
	err = app.cmd.Open(shell.MakeUniqueName(shell.Dir))
	if err != nil {
		return err
	}
	//启动websocket server
	app.ws = newServer(addr)
	app.ws.start()

	t := time.NewTicker(time.Second * 30)
Loop:
	for {
		select {
		case <-app.ctx.Done():
			break Loop
		case <-t.C:
			//%v: print value
			//%+v:print type:value
			//%#v:print struct{type:value...}
			glog.V(LVERBOSE).Infof("\n%s\nnumber of goroutine:%d\n",
				app.ws.State(),
				runtime.NumGoroutine())
		}
	}
	return nil
}

//结束应用程序
func (app *App) Quit() {
	app.cancel()
}

//某个请求完成，我们要通知网络模块
func (app *App) onCompleted(connId uint32, resp face.Response) {
	app.ws.send(connId, resp)
}

func (app *App) OnMessage(cid uint32, message string) {
	if strings.EqualFold(message, "quit") ||
		strings.EqualFold(message, "stop") {
		err := app.cmd.Write(cid, "app will quit")
		if err != nil {
			fmt.Printf("write shell message to client[%d] failed\n", cid)
		}
		app.Quit()
	}
}
