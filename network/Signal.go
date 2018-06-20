package network

import (
	"os/signal"
	"os"
	"syscall"
	"fmt"
)
var socketSignal chan os.Signal
func WaitSignal() {
	socketSignal = make(chan os.Signal, 1)
	signal.Notify(socketSignal, os.Kill, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	sig := <-socketSignal
	fmt.Sprintf("server exit. signal: [%s]", sig)
}