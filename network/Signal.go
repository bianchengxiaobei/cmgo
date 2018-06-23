package network

import (
	"os/signal"
	"os"
	"syscall"
	"github.com/bianchengxiaobei/cmgo/log4g"
)
var socketSignal chan os.Signal
func WaitSignal() {
	socketSignal = make(chan os.Signal, 1)
	signal.Notify(socketSignal, os.Kill, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	sig := <-socketSignal
	log4g.Infof("服务器退出!信号: [%s]", sig)
}