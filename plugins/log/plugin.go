package main

import (
	"fmt"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"sync"
)

var (
	lock *sync.Mutex = new(sync.Mutex)
)

func Reload() {
}

func Setup(ctx *plugins.PluginContext) {

}

func Auth(packet *radius.Packet) bool {
	write("auth", packet)
	return true
}

func Accounting(packet *radius.Packet) {
	write("accounting", packet)
}

func write(mode string, packet *radius.Packet) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		f, t := plugins.DatedFile(nil, mode)
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
