package main

import (
	"bufio"
	"fmt"
	"github.com/epiphyte/goutils"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"os"
	"strings"
)

type context struct {
	debug    bool
	secret   string
	preauths []plugins.PreAuth
	accts    []plugins.Accounting
	auths    []plugins.Authing
	// shortcuts
	preauth bool
	acct    bool
	auth    bool
}

func (ctx *context) authorize(buffer []byte) bool {
	valid := true
	if ctx.preauth || ctx.auth {
		p, err := ctx.packet(buffer)
		// we may not be able to always read a packet during conversation
		// especially during initial EAP phases
		// we let that go
		if err == nil {
			if ctx.preauth {
				for _, mod := range ctx.preauths {
					if mod.Pre(p) {
						continue
					}
					valid = false
					goutils.WriteDebug(fmt.Sprintf("unauthorized (failed: %s)", mod.Name()))
					break
				}
			}
			if ctx.auth {
				for _, mod := range ctx.auths {
					mod.Auth(p)
				}
			}
		}
	}
	return valid
}

func parseSecrets(secretFile string) string {
	if goutils.PathNotExists(secretFile) {
		panic("secrets file does not exist")
	}
	f, err := os.Open(secretFile)
	if logError("secret parsing", err) {
		panic("unable to read file for secrets")
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, "127.0.0.1") {
			parts := strings.Split(l, " ")
			return strings.TrimSpace(strings.Join(parts[1:], " "))
		}
	}
	panic("unable to find shared secret entry")
}

func (ctx *context) reload() {
	goutils.WriteInfo("received SIGINT")
	if ctx.preauth {
		for _, mod := range ctx.preauths {
			mod.Reload()
		}
	}
	if ctx.acct {
		for _, mod := range ctx.accts {
			mod.Reload()
		}
	}
}

func (ctx *context) packet(buffer []byte) (*radius.Packet, error) {
	return radius.Parse(buffer, []byte(ctx.secret))
}

func (ctx *context) account(buffer []byte) {
	p, e := ctx.packet(buffer)
	if e != nil {
		// unable to parse, exit early
		return
	}
	if ctx.acct {
		for _, mod := range ctx.accts {
			mod.Account(p)
		}
	}
}
