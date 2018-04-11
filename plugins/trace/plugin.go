package main

import (
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"log"
)

type tracer struct {
}

var Plugin tracer

func (t *tracer) Reload() {
}

func (t *tracer) Name() string {
	return "tracer"
}

func (t *tracer) Setup(ctx *plugins.PluginContext) {
}

func (t *tracer) Auth(packet *radius.Packet) bool {
	dump("auth", packet)
	return true
}

func (t *tracer) Account(packet *radius.Packet) {
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
