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

type App struct {
	ws     *server
	cmd    shell.Server
	ctx    context.Context
	cancel context.CancelFunc
}

func NewApp() *App {
	app := &App{}
	app.ctx, app.cancel = context.WithCancel(context.Background())
	return app
}

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
	face.GetFaceInstance().OnCompleted = app.onCompleted
	err := face.GetFaceInstance().Init()
	if err != nil {
		return err
	}

	app.cmd = shell.NewServer(app)
	err = app.cmd.Open(shell.MakeUniqueName(shell.Dir))
	if err != nil {
		return err
	}
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

func (app *App) Quit() {
	app.cancel()
}

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
