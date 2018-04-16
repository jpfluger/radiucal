package main

import (
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"log"
)

type tracer struct {
}

var (
	Plugin tracer
	modes  []string
)

func (t *tracer) Reload() {
}

func (t *tracer) Name() string {
	return "tracer"
}

func (t *tracer) Setup(ctx *plugins.PluginContext) {
	modes = plugins.DisabledModes(t, ctx)
}

func (t *tracer) Auth(packet *radius.Packet) {
	dump(plugins.AuthingMode, packet)
}

func (t *tracer) Account(packet *radius.Packet) {
	dump(plugins.AccountingMode, packet)
}

func dump(mode string, packet *radius.Packet) {
	go func() {
		if plugins.Disabled(mode, modes) {
			return
		}
		log.Println(mode)
		attr := plugins.KeyValueStrings(packet)
		for _, a := range attr {
			log.Println(a)
		}
	}()
}
