package main

import (
	"flag"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"net"
	"time"
)

func newPacket(user, mac string) []byte {
	var secret = []byte("secret")
	p := radius.New(radius.CodeAccessRequest, secret)
	if err := rfc2865.UserName_AddString(p, user); err != nil {
		panic("unable to set attribute: user-name")
	}
	if err := rfc2865.CallingStationID_AddString(p, mac); err != nil {
		panic("unable to set attribute: calling-station-id")
	}
	b, err := p.Encode()
	if err != nil {
		panic("unable to encode packet")
	}
	return b
}

func runEndpoint() {
	addr, err := net.ResolveUDPAddr("udp", ":1814")
	if err != nil {
		panic("unable to get address")
	}
	srv, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic("unable to listen")
	}
	for {
		var buffer []byte
		srv.Read(buffer)
	}
}

func write(user, mac string, conn *net.UDPConn) {
	time.Sleep(1 * time.Second)
	p := newPacket(user, mac)
	_, err := conn.Write(p)
	if err != nil {
		panic("unable to write")
	}
}

func test() {
	addr, err := net.ResolveUDPAddr("udp", ":1812")
	if err != nil {
		panic("unable to get address")
	}
	srv, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic("unable to dial")
	}
	write("test", "11-22-33-44-55-66", srv)
	write("test", "11-22-33-44-55-67", srv)
	write("test", "11-22-33-44-55-66", srv)
}

func main() {
	endpoint := flag.Bool("endpoint", false, "indicates if running as a fake endpoint")
	flag.Parse()
	if *endpoint {
		runEndpoint()
	} else {
		test()
	}
}
