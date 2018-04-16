package main

import (
	"fmt"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"sync"
)

var (
	lock     *sync.Mutex = new(sync.Mutex)
	logs     string
	Plugin   logger
	modes    []string
	instance string
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
	modes = plugins.DisabledModes(l, ctx)
	instance = ctx.Instance
}

func (l *logger) Pre(packet *radius.Packet) bool {
	write(plugins.PreAuthMode, packet)
	return true
}

func (l *logger) Auth(packet *radius.Packet) {
	write(plugins.AuthingMode, packet)
}

func (l *logger) Account(packet *radius.Packet) {
	write(plugins.AccountingMode, packet)
}

func write(mode string, packet *radius.Packet) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		if plugins.Disabled(mode, modes) {
			return
		}
		f, t := plugins.DatedAppendFile(logs, mode, instance)
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
