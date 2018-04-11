package plugins

import (
	"errors"
	"fmt"
	"layeh.com/radius"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"
	"unicode"
)

type PluginContext struct {
	// Enable debugging in plugins
	Debug bool
	// Allow plugins to cache data
	Cache bool
	// Location of logs directory
	Logs string
	// Location of the general lib directory
	Lib string
}

type Module interface {
	Reload()
	Setup(*PluginContext)
	Name() string
}

type PreAuth interface {
	Module
	Pre(*radius.Packet) bool
}

type Authing interface {
	Module
	Auth(*radius.Packet)
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

func DatedFile(path, name string) (*os.File, time.Time) {
	t := time.Now()
	logPath := filepath.Join(path, fmt.Sprintf("radiucal.%s.%s", name, t.Format("2006-01-02")))
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

func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func LoadPreAuthPlugin(path string, ctx *PluginContext) (PreAuth, error) {
	p, err := loadPlugin(path, ctx)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New(fmt.Sprintf("%s is not a preauth plugin"))
	}
	return p.(PreAuth), nil
}

func LoadAccountingPlugin(path string, ctx *PluginContext) (Accounting, error) {
	a, err := loadPlugin(path, ctx)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, errors.New(fmt.Sprintf("%s is not an accounting plugin"))
	}
	return a.(Accounting), nil
}

func loadPlugin(path string, ctx *PluginContext) (Module, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	v, err := p.Lookup("Plugin")
	if err != nil {
		return nil, err
	}
	var mod Module
	mod, ok := v.(Module)
	if !ok {
		return nil, errors.New(fmt.Sprintf("unable to load plugin %s", path))
	}
	mod.Setup(ctx)
	switch t := mod.(type) {
	default:
		return nil, errors.New(fmt.Sprintf("unknown type: %T", t))
	case Accounting:
		return t.(Accounting), nil
	case PreAuth:
		return t.(PreAuth), nil
	}
}
