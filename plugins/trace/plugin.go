package main

import (
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
)

func Reload() {
}

func Setup(ctx *plugins.PluginContext) {

}

func Auth(packet *radius.Packet) bool {
	return true
}

func Accounting(packet *radius.Packet) {
}
