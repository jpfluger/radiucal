// Implementation of a UDP proxy

package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"log"
)

const bSize = 1500
var (
	proxy *net.UDPConn
	serverAddress *net.UDPAddr
	clients map[string]*connection = make(map[string]*connection)
	mutex *sync.Mutex = new(sync.Mutex)
)

type connection struct {
	client *net.UDPAddr
	server *net.UDPConn
}

func logError(message string, err error) bool {
	if err == nil {
		return false
	}
	log.Println(fmt.Sprintf("[ERROR] %s", message))
	log.Println(err)
	return true;
}

func newConnection(srv, cli *net.UDPAddr) *connection {
	conn := new(connection)
	conn.client = cli
	srvudp, err := net.DialUDP("udp", nil, srv)
	if logError("dial udp", err) {
		return nil
	}
	conn.server = srvudp
	return conn
}


func setup(hostport string, port int) error {
	saddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	pudp, err := net.ListenUDP("udp", saddr)
	if err != nil {
		return err
	}
	proxy = pudp
	srvaddr, err := net.ResolveUDPAddr("udp", hostport)
	if err != nil {
		return err
	}
	serverAddress = srvaddr
	return nil
}

func runConnection(conn *connection) {
	var buffer [bSize]byte
	for {
		n, err := conn.server.Read(buffer[0:])
		if logError("unable to read", err) {
			continue
		}
		_, err = proxy.WriteToUDP(buffer[0:n], conn.client)
		logError("relaying", err)
	}
}

func audit(conn *connection) {
	runConnection(conn)
}

func runProxy() {
	var buffer [bSize]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if logError("read from udp", err) {
			continue
		}
		saddr := cliaddr.String()
		mutex.Lock()
		conn, found := clients[saddr]
		if !found {
			conn = newConnection(serverAddress, cliaddr)
			if conn == nil {
				mutex.Unlock()
				continue
			}
			clients[saddr] = conn
			mutex.Unlock()
			go audit(conn)
		} else {
			mutex.Unlock()
		}
		_, err = conn.server.Write(buffer[0:n])
		logError("server write", err)
	}
}

func main() {
	log.SetFlags(0)
	var from = flag.Int("from", 1812, "Proxy (from) port")
	var to = flag.Int("to", 1814, "Server (to) port")
	var host = flag.String("host", "localhost", "Server address")
	addr := fmt.Sprintf("%s:%d", *host, *to)
	err := setup(addr, *from)
	if logError("proxy setup", err) {
		panic("unable to proceed")
	}
	runProxy()
}
