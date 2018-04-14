package main

import (
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"testing"
)

type MockModule struct {
	acct int
	auth int
	pre  int
	fail bool
}

func (m *MockModule) Name() string {
	return "mock"
}

func (m *MockModule) Reload() {
}

func (m *MockModule) Setup(c *plugins.PluginContext) {
}

func (m *MockModule) Pre(p *radius.Packet) bool {
	m.pre++
	return !m.fail
}

func (m *MockModule) Auth(p *radius.Packet) {
	m.auth++
}

func (m *MockModule) Account(p *radius.Packet) {
	m.acct++
}

func TestAuthNoMods(t *testing.T) {
	ctx := &context{}
	if !ctx.authorize(nil) {
		t.Error("should have passed, nothing to do")
	}
}

func TestAuth(t *testing.T) {
	ctx, p := getPacket(t)
	m := &MockModule{}
	ctx.auths = append(ctx.auths, m)
	ctx.authorize(nil)
	if m.auth != 0 {
		t.Error("did auth")
	}
	ctx.authorize(p)
	if m.auth != 1 {
		t.Error("didn't auth")
	}
}

func getPacket(t *testing.T) (*context, []byte) {
	var secret = []byte("secret")
	p := radius.New(radius.CodeAccessRequest, secret)
	if err := rfc2865.UserName_AddString(p, "user"); err != nil {
		t.Error("unable to add user name")
	}
	if err := rfc2865.CallingStationID_AddString(p, "11-22-33-44-55-66"); err != nil {
		t.Error("unable to add calling station")
	}
	b, err := p.Encode()
	if err != nil {
		t.Error("unable to encode")
	}
	return &context{}, b
}

func TestSecretParsing(t *testing.T) {
}

func TestReload(t *testing.T) {
}

func TestAcctNoMods(t *testing.T) {
}

func TestAcct(t *testing.T) {
}
