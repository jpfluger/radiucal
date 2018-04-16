// Implementation of a UDP proxy

package main

import (
	"flag"
	"fmt"
	"github.com/epiphyte/goutils"
	"github.com/epiphyte/radiucal/plugins"
	"layeh.com/radius"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
)

var vers = "master"

var (
	proxy         *net.UDPConn
	serverAddress *net.UDPAddr
	clients       map[string]*connection = make(map[string]*connection)
	clientLock    *sync.Mutex            = new(sync.Mutex)
)

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
	var buffer [radius.MaxPacketLength]byte
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
	var buffer [radius.MaxPacketLength]byte
	for {
		n, cliaddr, err := proxy.ReadFromUDP(buffer[0:])
		if logError("read from udp", err) {
			continue
		}
		saddr := cliaddr.String()
		clientLock.Lock()
		conn, found := clients[saddr]
		if !found {
			conn = newConnection(serverAddress, cliaddr)
			if conn == nil {
				clientLock.Unlock()
				continue
			}
			clients[saddr] = conn
			clientLock.Unlock()
			go runConnection(conn)
		} else {
			clientLock.Unlock()
		}
		if !ctx.authorize([]byte(buffer[0:n])) {
			continue
		}
		_, err = conn.server.Write(buffer[0:n])
		logError("server write", err)
	}
}

func account(ctx *context) {
	var buffer [radius.MaxPacketLength]byte
	for {
		n, _, err := proxy.ReadFromUDP(buffer[0:])
		if logError("accounting udp error", err) {
			continue
		}
		ctx.account(buffer[0:n])
	}
}

func main() {
	goutils.WriteInfo(fmt.Sprintf("radiucal (%s)", vers))
	var config = flag.String("config", "/etc/radiucal/radiucal.conf", "Configuration file")
	var instance = flag.String("instance", "", "Instance name")
	var debugging = flag.Bool("debug", false, "debugging")
	flag.Parse()
	conf, err := goutils.LoadConfig(*config, goutils.NewConfigSettings())
	if err != nil {
		goutils.WriteError("unable to load config", err)
		panic("invalid/unable to load config")
	}
	debug := conf.GetTrue("debug") || *debugging
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
	pCtx.Logs = filepath.Join(lib, "log")
	pCtx.Lib = lib
	pCtx.Config = conf
	pCtx.Instance = *instance
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
		ctx.modules = append(ctx.modules, obj)
		ctx.module = true
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			clientLock.Lock()
			clients = make(map[string]*connection)
			clientLock.Unlock()
			ctx.reload()
		}
	}()

	if accounting {
		goutils.WriteInfo("accounting mode")
		account(ctx)
	} else {
		runProxy(ctx)
	}
}
