// Implementation of a UDP proxy

package main

import (
	"bufio"
	"flag"
	"fmt"
	"layeh.com/radius"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
)

const bSize = 1500
const lib = "/var/lib/radiucal/"

var vers = "master"

var (
	proxy         *net.UDPConn
	serverAddress *net.UDPAddr
	clients       map[string]*connection = make(map[string]*connection)
	mutex         *sync.Mutex            = new(sync.Mutex)
	markLock      *sync.Mutex            = new(sync.Mutex)
	auditLock     *sync.Mutex            = new(sync.Mutex)
	preLock       *sync.Mutex            = new(sync.Mutex)
	preauthed     map[string]bool        = make(map[string]bool)
)

type context struct {
	debug  bool
	secret string
}

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
	return true
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

func runProxy(ctx *context) {
	if ctx.debug {
		log.Println("=============WARNING==================")
		log.Println("debugging is enabled!")
		log.Println("dumps from debugging may contain secrets")
		log.Println("do NOT share debugging dumps")
		log.Println("=============WARNING==================")
		log.Println("secret")
		log.Println(ctx.secret)
	}
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
			go runConnection(conn)
		} else {
			mutex.Unlock()
		}
		// TODO: cut-in preauth plugins here
		_, err = conn.server.Write(buffer[0:n])
		logError("server write", err)
	}
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func parseSecrets(secretFile string) string {
	if !pathExists(secretFile) {
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

func reload(ctx *context) {
	log.Println("received SIGINT")
	// TODO: cut-in preauth reloads here
	// TODO: cut-in acct reloads here
}

func accounting(ctx *context) {
	var buffer [bSize]byte
	for {
		n, _, err := proxy.ReadFromUDP(buffer[0:])
		if logError("accounting udp error", err) {
			continue
		}

		_, err = radius.Parse([]byte(buffer[0:n]), []byte(ctx.secret))
		if err != nil {
			// unable to read/parse this packet so move on
			continue
		}
		// TODO: cut-in acct plugins here
	}
}

func main() {
	log.SetFlags(0)
	log.Println(fmt.Sprintf("radiucal (%s)", vers))
	var port = flag.Int("port", 1812, "Listening port")
	var to = flag.Int("to", 1814, "Server (to) port")
	var host = flag.String("host", "localhost", "Server address")
	var debug = flag.Bool("debug", false, "debug mode")
	var secrets = flag.String("secrets", lib+"secrets", "shared secret with hostapd")
	var acct = flag.Bool("accounting", false, "run as an account server")
	// TODO: configuration parsing and defaults
	flag.Parse()
	addr := fmt.Sprintf("%s:%d", *host, *to)
	err := setup(addr, *port)
	if logError("proxy setup", err) {
		panic("unable to proceed")
	}
	secret := parseSecrets(*secrets)
	ctx := &context{debug: *debug, secret: secret}
	if *acct {
		log.Println("accounting mode")
		accounting(ctx)
	} else {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for sig := range c {
				log.Println("captured:", sig)
				reload(ctx)
			}
		}()
		runProxy(ctx)
	}
}
