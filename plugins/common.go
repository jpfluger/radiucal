package plugins

import (
	"fmt"
	"layeh.com/radius"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type PluginContext struct {
	Debug bool
	Logs  string
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

func DatedFile(ctx *PluginContext, name string) (*os.File, time.Time) {
	t := time.Now()
	logPath := filepath.Join(ctx.Logs, fmt.Sprintf("radiucal.%s.%s", name, t.Format("2006-01-02")))
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		log.Println("unable to create file")
		log.Println(logPath)
		log.Println(err)
		return nil, t
	}
	return f, t
}

func FormatLog(f *os.File, t time.Time, indicator, message string) {
	f.Write([]byte(fmt.Sprintf("%s [%s] %s\n", t.Format("2006-01-02T15:04:05"), strings.ToUpper(indicator), message)))
}
