package main

import (
	"fmt"
	"github.com/epiphyte/radiucal/plugins"
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
	return fmt.Sprintf("first: %s\nlast: %s\ncount: %d\n name: %s\n",
		m.first.Format("2006-01-02T15:04:05"),
		m.last.Format("2006-01-02T15:04:05"),
		m.count,
		m.name)
}

var (
	lock   *sync.Mutex = new(sync.Mutex)
	dir    string
	Plugin stats
	info   map[string]*modedata = make(map[string]*modedata)
)

type stats struct {
}

func (s *stats) Name() string {
	return "stats"
}

func (s *stats) Reload() {
}

func (s *stats) Setup(ctx *plugins.PluginContext) {
	dir = ctx.Logs
}

func (s *stats) Pre(packet *radius.Packet) bool {
	write("preauth")
	return true
}

func (s *stats) Auth(packet *radius.Packet) {
	write("auth")
}

func (s *stats) Account(packet *radius.Packet) {
	write("accounting")
}

func write(mode string) {
	go func() {
		lock.Lock()
		defer lock.Unlock()
		f, t := plugins.DatedFile(dir, fmt.Sprintf("stats.%s", mode))
		if f == nil {
			return
		}
		if _, ok := info[mode]; !ok {
			info[mode] = &modedata{first: t, count: 0, name: mode}
		}
		m, _ := info[mode]
		m.last = t
		m.count++
		f.Write([]byte(m.String()))
	}()
}
