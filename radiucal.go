// Implementation of a UDP proxy

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/epiphyte/goutils"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
)

const bSize = 1500

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
	auths    []plugins.Authing
	// shortcuts
	preauth bool
	acct    bool
	auth    bool
}

type connection struct {
	client *net.UDPAddr
	server *net.UDPConn
}

func logError(message string, err error) bool {
	if err == nil {
		return false
	}
	goutils.WriteError(message, err)
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
		goutils.WriteInfo("=============WARNING==================")
		goutils.WriteInfo("debugging is enabled!")
		goutils.WriteInfo("dumps from debugging may contain secrets")
		goutils.WriteInfo("do NOT share debugging dumps")
		goutils.WriteInfo("=============WARNING==================")
		goutils.WriteDebug("secret", ctx.secret)
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
		if ctx.preauth || ctx.auth {
			p, err := radius.Parse([]byte(buffer[0:n]), []byte(ctx.secret))
			// we may not be able to always read a packet during conversation
			// especially during initial EAP phases
			// we let that go
			if err == nil {
				valid := true
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
	goutils.WriteInfo(fmt.Sprintf("radiucal (%s)", vers))
	var config = flag.String("config", "/etc/radiucal/radiucal.conf", "Configuration file")
	var instance = flag.String("instance", "", "Instance name")
	flag.Parse()
	conf, err := goutils.LoadConfig(*config, goutils.NewConfigSettings())
	if err != nil {
		goutils.WriteError("unable to load config", err)
		panic("invalid/unable to load config")
	}
	debug := conf.GetTrue("debug")
	logOpts := goutils.NewLogOptions()
	logOpts.Debug = debug
	logOpts.Info = true
	logOpts.Instance = *instance
	goutils.ConfigureLogging(logOpts)
	host := conf.GetStringOrDefault("host", "localhost")
	var to int = 1814
	accounting := conf.GetTrue("accounting")
	defaultBind := 1812
	if accounting {
		defaultBind = 1813
	} else {
		to, err = conf.GetIntOrDefault("to", 1814)
		if err != nil {
			goutils.WriteError("unable to get bind-to", err)
			panic("cannot bind to another socket")
		}
	}
	bind, err := conf.GetIntOrDefault("bind", defaultBind)
	if err != nil {
		goutils.WriteError("unable to bind address", err)
		panic("unable to bind")
	}
	addr := fmt.Sprintf("%s:%d", host, to)
	err = setup(addr, bind)
	if logError("proxy setup", err) {
		panic("unable to proceed")
	}

	lib := conf.GetStringOrDefault("dir", "/var/lib/radiucal/")
	secrets := filepath.Join(lib, "secrets")
	secret := parseSecrets(secrets)
	ctx := &context{debug: debug, secret: secret}
	mods := conf.GetArrayOrEmpty("plugins")
	pCtx := &plugins.PluginContext{}
	pCtx.Cache = conf.GetTrue("cache")
	pCtx.Logs = filepath.Join(lib, "log")
	pCtx.Lib = lib
	pPath := filepath.Join(lib, "plugins")
	for _, p := range mods {
		oPath := filepath.Join(pPath, fmt.Sprintf("%s.rd", p))
		goutils.WriteInfo("loading plugin", p, oPath)
		obj, err := plugins.LoadPlugin(oPath, pCtx)
		if err != nil {
			goutils.WriteError(fmt.Sprintf("unable to load plugin: %s", p), err)
			panic("unable to load plugin")
		}
		if i, ok := obj.(plugins.Accounting); ok {
			ctx.acct = true
			ctx.accts = append(ctx.accts, i)
		}
		if i, ok := obj.(plugins.Authing); ok {
			ctx.auth = true
			ctx.auths = append(ctx.auths, i)
		}
		if i, ok := obj.(plugins.PreAuth); ok {
			ctx.preauth = true
			ctx.preauths = append(ctx.preauths, i)
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			reload(ctx)
		}
	}()

	if accounting {
		goutils.WriteInfo("accounting mode")
		account(ctx)
	} else {
		runProxy(ctx)
	}
}
