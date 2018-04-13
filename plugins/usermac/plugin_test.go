package main

import (
	"testing"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"github.com/epiphyte/goutils"
)

func TestUserMacBasics(t *testing.T) {
	newTestSet(t, "test", "11-22-33-44-55-66", true)
	newTestSet(t, "test", "12-22-33-44-55-66", false)
}

func newTestSet(t *testing.T, user, mac string, valid bool) (*radius.Packet, *umac) {
	m := setupUserMac()
	if m.Name() != "usermac" {
		t.Error("invalid/wrong name")
	}
	var secret = []byte("secret")
	p := radius.New(radius.CodeAccessRequest, secret)
	if m.Pre(p) {
		t.Error("no username/calling station id")
	}
	if err := rfc2865.UserName_AddString(p, user); err != nil {
		t.Error("unable to add user name")
	}
	if m.Pre(p) {
		t.Error("no calling station set")
	}
	if err := rfc2865.CallingStationID_AddString(p, mac); err != nil {
		t.Error("unable to add calling station")
	}
	if valid && !m.Pre(p) {
		t.Error("failed auth test")
	}
	if !valid && m.Pre(p) {
		t.Error("should have failed auth test")
	}
	return p, m
}

func setupUserMac() *umac {
	opts := goutils.NewLogOptions()
	opts.Debug = true
	opts.Info = true
	goutils.ConfigureLogging(opts)
	canCache = true
	logs = "./tests/"
	db = "./tests/"
	return &umac{}
}

func TestUserMacCache(t *testing.T) {
	i := 0
	for i <= 1 {
		i++
	}
}

func TestUserMacCallback(t *testing.T) {
}

func TestUserMacLog(t *testing.T) {
}
