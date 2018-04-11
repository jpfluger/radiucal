package plugins

import (
	"layeh.com/radius"
)

type PluginContext struct {
	Debug bool
}

type Module interface {
	Reload()
	Setup(*PluginContext)
}

type PreAuth interface {
	Module
	Auth(*radius.Packet) bool
}

type Accounting interface {
	Module
	Account(*radius.Packet)
}
