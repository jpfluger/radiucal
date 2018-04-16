package plugins

import (
	"errors"
	"fmt"
	"github.com/epiphyte/goutils"
	"layeh.com/radius"
	. "layeh.com/radius/rfc2865"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"
	"unicode"
)

const (
	AccountingMode = "accounting"
	AuthingMode    = "auth"
	PreAuthMode    = "preauth"
)

type PluginContext struct {
	// Location of logs directory
	Logs string
	// Location of the general lib directory
	Lib string
	// Backing config
	Config *goutils.Config
	// Instance name
	Instance string
}

type Module interface {
	Reload()
	Setup(*PluginContext)
	Name() string
}

type PreAuth interface {
	Module
	Pre(*radius.Packet) bool
}

type Authing interface {
	Module
	Auth(*radius.Packet)
}

type Accounting interface {
	Module
	Account(*radius.Packet)
}

// Get attributes as Type/Value string arrays
func KeyValueStrings(packet *radius.Packet) []string {
	var datum []string
	for t, a := range packet.Attributes {
		name := resolveType(t)
		datum = append(datum, fmt.Sprintf("Type: %d (%s)", t, name))
		for _, s := range a {
			unknown := true
			val := ""
			if t == NASIPAddress_Type {
				ip, err := radius.IPAddr(s)
				if err == nil {
					unknown = false
					val = fmt.Sprintf("(ip) %s", ip.String())
				}
			}

			if unknown {
				i, err := radius.Integer(s)
				if err == nil {
					unknown = false
					val = fmt.Sprintf("(int) %d", i)
				}
			}

			if unknown {
				d, err := radius.Date(s)
				if err == nil {
					unknown = false
					val = fmt.Sprintf("(time) %s", d.Format(time.RFC3339))
				}
			}

			if unknown {
				val = string(s)
				unknown = false
				for _, c := range val {
					if !unicode.IsPrint(c) {
						unknown = true
						break
					}
				}
			}

			if unknown {
				val = fmt.Sprintf("(hex) %x", s)
			}
			datum = append(datum, fmt.Sprintf("Value: %s", val))
		}
	}
	return datum
}

func DatedAppendFile(path, name string) (*os.File, time.Time) {
	return newFile(path, name, true)
}

func NewFilePath(path, name string) (string, time.Time) {
	t := time.Now()
	logPath := filepath.Join(path, fmt.Sprintf("radiucal.%s.%s", name, t.Format("2006-01-02")))
	return logPath, t
}

func Disabled(mode string, modes []string) bool {
	if len(modes) == 0 {
		return false
	}
	for _, m := range modes {
		if m == mode {
			return true
		}
	}
	return false
}

func DisabledModes(m Module, ctx *PluginContext) []string {
	name := m.Name()
	accounting := ctx.Config.GetTrue(fmt.Sprintf("%s_disable_accounting", name))
	authing := ctx.Config.GetTrue(fmt.Sprintf("%s_disable_auth", name))
	preauth := ctx.Config.GetTrue(fmt.Sprintf("%s_disable_preauth", name))
	var modes []string
	if accounting {
		modes = append(modes, AccountingMode)
	}
	if authing {
		modes = append(modes, AuthingMode)
	}
	if preauth {
		modes = append(modes, PreAuthMode)
	}
	return modes
}

func newFile(path, name string, appending bool) (*os.File, time.Time) {
	flags := os.O_RDWR | os.O_CREATE
	if appending {
		flags = flags | os.O_APPEND
	}
	logPath, t := NewFilePath(path, name)
	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		goutils.WriteError(fmt.Sprintf("unable to create file: %s", logPath), err)
		return nil, t
	}
	return f, t
}

func FormatLog(f *os.File, t time.Time, indicator, message string) {
	f.Write([]byte(fmt.Sprintf("%s [%s] %s\n", t.Format("2006-01-02T15:04:05"), strings.ToUpper(indicator), message)))
}

func LoadPlugin(path string, ctx *PluginContext) (Module, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	v, err := p.Lookup("Plugin")
	if err != nil {
		return nil, err
	}
	var mod Module
	mod, ok := v.(Module)
	if !ok {
		return nil, errors.New(fmt.Sprintf("unable to load plugin %s", path))
	}
	mod.Setup(ctx)
	return mod, nil
	switch t := mod.(type) {
	default:
		return nil, errors.New(fmt.Sprintf("unknown type: %T", t))
	case Accounting:
		return t.(Accounting), nil
	case PreAuth:
		return t.(PreAuth), nil
	case Authing:
		return t.(Authing), nil
	}
}

// rfc2865 only, get string names for types
func resolveType(t radius.Type) string {
	switch t {
	case UserName_Type:
		return "User-Name"
	case UserPassword_Type:
		return "User-Password"
	case CHAPPassword_Type:
		return "CHAP-Password"
	case NASIPAddress_Type:
		return "NAS-IP-Address"
	case NASPort_Type:
		return "NAS-Port"
	case ServiceType_Type:
		return "Service-Type"
	case FramedProtocol_Type:
		return "Framed-Protocol"
	case FramedIPAddress_Type:
		return "Framed-IP-Address"
	case FramedIPNetmask_Type:
		return "Framed-IP-Netmask"
	case FramedRouting_Type:
		return "Framed-Routing"
	case FilterID_Type:
		return "Filter-ID"
	case FramedMTU_Type:
		return "Framed-MTU"
	case FramedCompression_Type:
		return "Framed-Compression"
	case LoginIPHost_Type:
		return "Login-IP-Host"
	case LoginService_Type:
		return "Login-Service"
	case LoginTCPPort_Type:
		return "Login-TCP-Port"
	case ReplyMessage_Type:
		return "Reply-Message"
	case CallbackNumber_Type:
		return "Callback-Number"
	case CallbackID_Type:
		return "Callback-ID"
	case FramedRoute_Type:
		return "Framed-Route"
	case FramedIPXNetwork_Type:
		return "Framed-IPX-Network"
	case State_Type:
		return "State"
	case Class_Type:
		return "Class"
	case VendorSpecific_Type:
		return "Vendor-Specific"
	case SessionTimeout_Type:
		return "Session-Timeout"
	case IdleTimeout_Type:
		return "Idle-Timeout"
	case TerminationAction_Type:
		return "Termination-Action"
	case CalledStationID_Type:
		return "Called-Station-ID"
	case CallingStationID_Type:
		return "Calling-Station-ID"
	case NASIdentifier_Type:
		return "NAS-Identifier"
	case ProxyState_Type:
		return "Proxy-State"
	case LoginLATService_Type:
		return "Login-LAT-Service"
	case LoginLATNode_Type:
		return "Login-LAT-Node"
	case LoginLATGroup_Type:
		return "Login-LAT-Group"
	case FramedAppleTalkLink_Type:
		return "Framed-Apple-Talk-Link"
	case FramedAppleTalkNetwork_Type:
		return "Framed-Apple-Talk-Network"
	case FramedAppleTalkZone_Type:
		return "Framed-Apple-Talk-Zone"
	case CHAPChallenge_Type:
		return "CHAP-Challenge"
	case NASPortType_Type:
		return "NAS-Port-Type"
	case PortLimit_Type:
		return "Port-Limit"
	case LoginLATPort_Type:
		return "Login-LAT-Port"
	}
	return "Unknown"
}
