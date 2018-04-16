package main

import (
	"fmt"
	"github.com/epiphyte/radiucal/plugins"
	"io/ioutil"
	"layeh.com/radius"
	"sync"
	"time"
)

type modedata struct {
	first time.Time
	last  time.Time
	name  string
	count int
}

func (m *modedata) String() string {
	return fmt.Sprintf("first: %s\nlast: %s\ncount: %d\nname: %s\n",
		m.first.Format("2006-01-02T15:04:05"),
		m.last.Format("2006-01-02T15:04:05"),
		m.count,
		m.name)
}

var (
	lock     *sync.Mutex = new(sync.Mutex)
	dir      string
	Plugin   stats
	info     map[string]*modedata = make(map[string]*modedata)
	modes    []string
	instance string
)

type stats struct {
}

func (s *stats) Name() string {
	return "stats"
}

func (s *stats) Reload() {
	lock.Lock()
	defer lock.Unlock()
	info = make(map[string]*modedata)
}

func (s *stats) Setup(ctx *plugins.PluginContext) {
	dir = ctx.Logs
	instance = ctx.Instance
	modes = plugins.DisabledModes(s, ctx)
}

func (s *stats) Pre(packet *radius.Packet) bool {
	write(plugins.PreAuthMode)
	return true
}

func (s *stats) Auth(packet *radius.Packet) {
	write(plugins.AuthingMode)
}

func (s *stats) Account(packet *radius.Packet) {
	write(plugins.AccountingMode)
}

func write(mode string) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		if plugins.Disabled(mode, modes) {
			return
		}
		f, t := plugins.NewFilePath(dir, fmt.Sprintf("stats.%s", mode), instance)
		if _, ok := info[mode]; !ok {
			info[mode] = &modedata{first: t, count: 0, name: mode}
		}
		m, _ := info[mode]
		m.last = t
		m.count++
		ioutil.WriteFile(f, []byte(m.String()), 0644)
	}()
}
