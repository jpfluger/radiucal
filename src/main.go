// Implementation of a UDP proxy

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/epiphyte/radiucal/plugins"
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
)

type context struct {
	debug    bool
	secret   string
	preauths []plugins.PreAuth
	accts    []plugins.Accounting
	// shortcuts
	preauth bool
	acct    bool
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
		if ctx.preauth {
			p, err := radius.Parse([]byte(buffer[0:n]), []byte(ctx.secret))
			// we may not be able to always read a packet during conversation
			// especially during initial EAP phases
			// we let that go
			if err == nil {
				valid := true
				for _, mod := range ctx.preauths {
					if mod.Auth(p) {
						continue
					}
					valid = false
					log.Println(fmt.Sprintf("unauthorized (failed: %s)", mod.Name()))
					break
				}
				if !valid {
					continue
				}
			}
		}
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

func account(ctx *context) {
	var buffer [bSize]byte
	for {
		n, _, err := proxy.ReadFromUDP(buffer[0:])
		if logError("accounting udp error", err) {
			continue
		}

		p, err := radius.Parse([]byte(buffer[0:n]), []byte(ctx.secret))
		if err != nil {
			// unable to read/parse this packet so move on
			continue
		}
		if ctx.acct {
			for _, mod := range ctx.accts {
				mod.Account(p)
			}
		}
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

	// TODO: until we're ready to switch to configs
	pAcct := []string{"log", "trace"}
	pAuth := []string{"log", "trace", "usermac"}
	pCtx := &plugins.PluginContext{}
	pCtx.Debug = ctx.debug
	pCtx.Cache = true
	pCtx.Logs = lib + "logs"
	pCtx.Lib = lib
	for _, a := range pAuth {
		ctx.preauth = true
		mod, err := plugins.LoadPreAuthPlugin(lib+"plugins/"+a+".so", pCtx)
		if err != nil {
			log.Println("unable to load preauth plugin")
			log.Println(err)
			panic("unable to load plugin")
		}
		log.Println("loaded", mod.Name())
		ctx.preauths = append(ctx.preauths, mod)
	}
	for _, a := range pAcct {
		ctx.acct = true
		mod, err := plugins.LoadAccountingPlugin(lib+"plugins/"+a+".so", pCtx)
		if err != nil {
			log.Println("unable to load accounting plugin")
			log.Println(err)
			panic("unable to load plugin")
		}
		log.Println("loaded", mod.Name())
		ctx.accts = append(ctx.accts, mod)
	}
	// TODO: end ^ todo

	if *acct {
		log.Println("accounting mode")
		account(ctx)
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
