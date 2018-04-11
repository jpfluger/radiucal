package main

import (
	"errors"
	"fmt"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	. "layeh.com/radius/rfc2865"
	"log"
	"path/filepath"
	"strings"
	"sync"
)

var (
	cache    map[string]bool = make(map[string]bool)
	lock     *sync.Mutex     = new(sync.Mutex)
	fileLock *sync.Mutex     = new(sync.Mutex)
	canCache bool
	debug    bool
	db       string
	logs     string
)

func Reload() {
}

func Setup(ctx *plugins.PluginContext) {

}

func Auth(packet *radius.Packet) bool {
	return true
}

func clean(in string) string {
	result := ""
	for _, c := range strings.ToLower(in) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' {
			result = result + string(c)
		}
	}
	return result
}

func attributesUserMac(p *radius.Packet) error {
	username, err := UserName_LookupString(p)
	if err != nil {
		return err
	}
	calling, err := CallingStationID_LookupString(p)
	if err != nil {
		return err
	}
	username = clean(username)
	calling = clean(calling)
	fqdn := fmt.Sprintf("%s.%s", username, calling)
	lock.Lock()
	good, ok := cache[fqdn]
	lock.Unlock()
	if canCache && ok {
		if debug {
			log.Println("object is preauthed")
		}
		if good {
			return nil
		} else {
			return errors.New(fmt.Sprintf("%s is blacklisted", fqdn))
		}
	}
	path := filepath.Join(db, fqdn)
	result := "passed"
	var failure error
	res := plugins.PathExists(path)
	lock.Lock()
	cache[fqdn] = res
	lock.Unlock()
	if !res {
		failure = errors.New(fmt.Sprintf("failed preauth: %s %s", username, calling))
		result = "failed"
	}
	go mark(result, username, calling, p)
	return failure
}

func mark(result, user, calling string, p *radius.Packet) {
	nas := clean(NASIdentifier_GetString(p))
	if len(nas) == 0 {
		nas = "unknown"
	}
	fileLock.Lock()
	defer fileLock.Unlock()
	f, t := plugins.DatedFile(logs, "audit")
	if f == nil {
		return
	}
	defer f.Close()
	plugins.FormatLog(f, t, result, fmt.Sprintf("%s (mac:%s) (nas:%s)", user, calling, nas))
}
