package main

import (
	"fmt"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"sync"
)

var (
	lock   *sync.Mutex = new(sync.Mutex)
	logs   string
	Plugin logger
)

type logger struct {
}

func (l *logger) Name() string {
	return "logger"
}

func (l *logger) Reload() {
}

func (l *logger) Setup(ctx *plugins.PluginContext) {
	logs = ctx.Logs
}

func (l *logger) Auth(packet *radius.Packet) {
	write("auth", packet)
}

func (l *logger) Account(packet *radius.Packet) {
	write("accounting", packet)
}

func write(mode string, packet *radius.Packet) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		f, t := plugins.DatedFile(logs, mode)
		if f == nil {
			return
		}
		attr := plugins.KeyValueStrings(packet)
		output := fmt.Sprintf("id -> %s \n", mode)
		plugins.FormatLog(f, t, mode, output)
		for _, a := range attr {
			output = output + fmt.Sprintf("%s\n", a)
			plugins.FormatLog(f, t, mode, output)
		}
	}()
}
