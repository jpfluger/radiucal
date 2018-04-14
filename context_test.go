package main

import (
	"testing"
)

func TestAuthNoMods(t *testing.T) {
	ctx := &context{}
	if !ctx.authorize(nil) {
		t.Error("should have passed, nothing to do")
	}
}

func TestAuth(t *testing.T) {
}

func TestSecretParsing(t *testing.T) {
}

func TestReload(t *testing.T) {
}

func TestAcctNoMods(t *testing.T) {
}

func TestAcct(t *testing.T) {
}
