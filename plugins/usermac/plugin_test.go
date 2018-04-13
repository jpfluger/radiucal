package main

import (
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"testing"
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
	canCache = true
	doCallback = false
	callback = []string{}
	logs = "./tests/"
	db = "./tests/"
	m := &umac{}
	m.Reload()
	return m
}

func TestUserMacCache(t *testing.T) {
	pg, m := newTestSet(t, "test", "11-22-33-44-55-66", true)
	pb, _ := newTestSet(t, "test", "11-22-33-44-55-68", false)
	for _, b := range []bool{true, false} {
		canCache = b
		if !m.Pre(pg) {
			t.Error("should re-auth")
		}
		if m.Pre(pb) {
			t.Error("should be blacklisted")
		}
	}
}

func TestUserMacCallback(t *testing.T) {
	p, m := newTestSet(t, "test", "11-22-33-44-55-66", true)
	newTestSet(t, "test", "12-22-33-44-55-66", false)
	canCache = false
	callback = []string{"echo"}
	doCallback = true
	if !m.Pre(p) {
		t.Error("should have authed")
	}
}
