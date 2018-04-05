// Implementation of a UDP proxy

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"layeh.com/radius"
	. "layeh.com/radius/rfc2865"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
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
)

type authmode struct {
	log     bool
	enabled bool
}

type context struct {
	logs    string
	db      string
	debug   bool
	preauth *authmode
	secret  string
	audit   bool
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

func clean(in string) string {
	result := ""
	for _, c := range strings.ToLower(in) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' {
			result = result + string(c)
		}
	}
	return result
}

func preauth(b string, ctx *context) error {
	p, err := radius.Parse([]byte(b), []byte(ctx.secret))
	if err != nil {
		// we can either parse or not understand
		// if we don't understand there is nothing to look at anyway
		return nil
	}
	username, err := UserName_LookupString(p)
	if err != nil {
		return err
	}
	calling, err := CallingStationID_LookupString(p)
	if err != nil {
		return err
	}
	username = clean(username)
	calling = clean(calling)
	path := filepath.Join(ctx.db, fmt.Sprintf("%s.%s", username, calling))
	result := "passed"
	var failure error
	res := pathExists(path)
	if !res {
		failure = errors.New(fmt.Sprintf("failed preauth: %s %s", username, calling))
		result = "failed"
	}
	if ctx.audit {
		go auditLog("auth", ctx, p)
	}
	if ctx.debug {
		go dump(ctx, p, printDump)
	}
	if ctx.preauth.log {
		go mark(ctx, result, username, calling, p)
	}
	return failure
}

func auditLog(id string, ctx *context, p *radius.Packet) {
	auditLock.Lock()
	defer auditLock.Unlock()
	f, t := getFile(ctx, id)
	if f == nil {
		return
	}
	fxn := func(data []string) {
		output := ""
		for _, d := range data {
			output = output + fmt.Sprintf("%s\n", d)
		}
		formatLog(f, t, id, output)
	}
	dump(ctx, p, fxn)
}

func printDump(data []string) {
	for _, d := range data {
		log.Println(d)
	}
}

type dumpCallback func(data []string)

func dump(ctx *context, p *radius.Packet, callback dumpCallback) {
	var datum []string
	for t, a := range p.Attributes {
		datum = append(datum, fmt.Sprintf("Type: %d", t))
		for _, s := range a {
			str := true
			val := string(s)
			for _, c := range val {
				if !unicode.IsPrint(c) {
					str = false
					break
				}
			}
			if !str {
				val = fmt.Sprintf("%x", s)
			}
			datum = append(datum, fmt.Sprintf("Value: %s", val))
		}
	}
	if len(datum) > 0 {
		callback(datum)
	}
}

func formatLog(f *os.File, t time.Time, indicator, message string) {
	f.Write([]byte(fmt.Sprintf("%s [%s] %s\n", t.Format("2006-01-02T15:04:05"), strings.ToUpper(indicator), message)))
}

func getFile(ctx *context, name string) (*os.File, time.Time) {
	t := time.Now()
	logPath := filepath.Join(ctx.logs, fmt.Sprintf("radiucal.%s.%s", t.Format("2006-01-02")))
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if logError(fmt.Sprintf("logging: %s", name), err) {
		return nil, t
	}
	return f, t
}

func mark(ctx *context, result, user, calling string, p *radius.Packet) {
	nas := clean(NASIdentifier_GetString(p))
	if len(nas) == 0 {
		nas = "unknown"
	}
	markLock.Lock()
	defer markLock.Unlock()
	f, t := getFile(ctx, "audit")
	if f == nil {
		return
	}
	defer f.Close()
	formatLog(f, t, result, fmt.Sprintf("%s (mac:%s) (nas:%s)", user, calling, nas))
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
		if ctx.preauth.enabled {
			audit := string(buffer[:n])
			err = preauth(audit, ctx)
			if err != nil {
				continue
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

func main() {
	log.SetFlags(0)
	log.Println(fmt.Sprintf("radiucal (%s)", vers))
	var from = flag.Int("from", 1812, "Proxy (from) port")
	var to = flag.Int("to", 1814, "Server (to) port")
	var host = flag.String("host", "localhost", "Server address")
	var db = flag.String("db", lib+"users/", "user.mac directory")
	var log = flag.String("log", lib+"log/", "audit logging")
	var debug = flag.Bool("debug", false, "debug mode")
	var pre = flag.Bool("preauth", true, "preauth checks")
	var preLog = flag.Bool("preauth-log", true, "preauth logging")
	var secrets = flag.String("secrets", lib+"secrets", "shared secret with hostapd")
	flag.Parse()
	if !pathExists(*db) || !pathExists(*log) {
		panic("missing required directory")
	}
	addr := fmt.Sprintf("%s:%d", *host, *to)
	err := setup(addr, *from)
	if logError("proxy setup", err) {
		panic("unable to proceed")
	}
	secret := parseSecrets(*secrets)
	preauthing := &authmode{enabled: *pre, log: *preLog}
	runProxy(&context{db: *db, logs: *log, debug: *debug, preauth: preauthing, secret: secret})
}
