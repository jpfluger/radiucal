package plugins

import (
	"fmt"
	"layeh.com/radius"
	"unicode"
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

// Get attributes as Type/Value string arrays
func KeyValueStrings(packet *radius.Packet) []string {
	var datum []string
	for t, a := range packet.Attributes {
		datum = append(datum, fmt.Sprintf("Type: %d", t))
		for _, s := range a {
			str := true
			val := string(s)
			for _, c := range val {
				if !unicode.IsPrint(c) {
					str = false
					break
				}
			}
			if !str {
				val = fmt.Sprintf("(hex) %x", s)
			}
			datum = append(datum, fmt.Sprintf("Value: %s", val))
		}
	}
	return datum
}
