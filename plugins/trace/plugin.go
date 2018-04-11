package main

import (
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"log"
)

func Reload() {
}

func Setup(ctx *plugins.PluginContext) {

}

func Auth(packet *radius.Packet) bool {
	dump("auth", packet)
	return true
}

func Accounting(packet *radius.Packet) {
	dump("accounting", packet)
}

func dump(mode string, packet *radius.Packet) {
	go func() {
		log.Println(mode)
		attr := plugins.KeyValueStrings(packet)
		for _, a := range attr {
			log.Println(a)
		}
	}()
}
