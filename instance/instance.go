// Copyright (C) 2021 github.com/V4NSH4J
//
// This source code has been released under the GNU Affero General Public
// License v3.0. A copy of this license is available at
// https://www.gnu.org/licenses/agpl-3.0.en.html

package instance

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/V4NSH4J/discord-mass-dm-GO/utilities"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type Instance struct {
	Token           string
	Email           string
	Password        string
	Proxy           string
	Cookie          string
	Fingerprint     string
	Messages        []Message
	Count           int
	LastQuery       string
	LastCount       int
	Members         []User
	AllMembers      []User
	Retry           int
	ScrapeCount     int
	ID              string
	Receiver        bool
	Config          Config
	GatewayProxy    string
	Client          *http.Client
	WG              *sync.WaitGroup
	Ws              *Connection
	fatal           chan error
	Invited         bool
	TimeServerCheck time.Time
	ChangedName     bool
	ChangedAvatar   bool
	LastID          int
	LastIDstr       string
	Mode            int
	UserAgent       string
	XSuper          string
}

func (in *Instance) StartWS() error {
	ws, err := in.NewConnection(in.wsFatalHandler)
	if err != nil {
		return fmt.Errorf("failed to create websocket connection: %s", err)
	}
	in.Ws = ws
	return nil
}

func (in *Instance) wsFatalHandler(err error) {
	if closeErr, ok := err.(*websocket.CloseError); ok && closeErr.Code == 4004 {
		in.fatal <- fmt.Errorf("websocket closed: authentication failed, try using a new token")
		return
	}
	color.Red("Websocket closed %v %v", err, in.Token)
	in.Receiver = false
	in.Ws, err = in.NewConnection(in.wsFatalHandler)
	if err != nil {
		in.fatal <- fmt.Errorf("failed to create websocket connection: %s", err)
		return
	}
	color.Green("Reconnected To Websocket")
}

func GetEverything() (Config, []Instance, error) {
	var cfg Config
	var instances []Instance
	var err error
	var tokens []string
	var proxies []string
	var proxy string
	var xsuper string
	var ua string

	// Load config
	cfg, err = GetConfig()
	if err != nil {
		return cfg, instances, err
	}
	supportedProtocols := []string{"http", "https", "socks4", "socks5"}
	if cfg.ProxySettings.ProxyProtocol != "" && !utilities.Contains(supportedProtocols, cfg.ProxySettings.ProxyProtocol) {
		color.Red("[!] You're using an unsupported proxy protocol. Assuming http by default")
		cfg.ProxySettings.ProxyProtocol = "http"
	}
	if cfg.ProxySettings.ProxyProtocol == "https" {
		cfg.ProxySettings.ProxyProtocol = "http"
	}
	if cfg.CaptchaSettings.CaptchaAPI == "" {
		color.Red("[!] You're not using a Captcha API, some functionality like invite joining might be unavailable")
	}
	if cfg.ProxySettings.Proxy != "" && os.Getenv("HTTPS_PROXY") == "" {
		os.Setenv("HTTPS_PROXY", cfg.ProxySettings.ProxyProtocol+"://"+cfg.ProxySettings.Proxy)
	}
	if !cfg.ProxySettings.ProxyFromFile && cfg.ProxySettings.ProxyForCaptcha {
		color.Red("[!] You must enabe proxy_from_file to use proxy_for_captcha")
		cfg.ProxySettings.ProxyForCaptcha = false
	}
	if cfg.CaptchaSettings.CaptchaAPI == "capcat.xyz" {
		color.Red("[!] You're using a third party Captcha solution, proceed with caution.")
	}
	if cfg.OtherSettings.Mode == 1 {
		// Discord App
		ua, xsuper = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) discord/0.0.61 Chrome/91.0.4472.164 Electron/13.6.6 Safari/537.36", "eyJvcyI6Ik1hYyBPUyBYIiwiYnJvd3NlciI6IkRpc2NvcmQgQ2xpZW50IiwicmVsZWFzZV9jaGFubmVsIjoicHRiIiwiY2xpZW50X3ZlcnNpb24iOiIwLjAuNjEiLCJvc192ZXJzaW9uIjoiMjEuNC4wIiwib3NfYXJjaCI6ImFybTY0Iiwic3lzdGVtX2xvY2FsZSI6ImVuLVVTIiwiY2xpZW50X2J1aWxkX251bWJlciI6MTI3MzA3LCJjbGllbnRfZXZlbnRfc291cmNlIjpudWxsfQ=="
	} else if cfg.OtherSettings.Mode == 2 {
		// Mobile
		ua, xsuper = "Discord/32249 CFNetwork/1331.0.7 Darwin/21.4.0", "eyJvcyI6ImlPUyIsImJyb3dzZXIiOiJEaXNjb3JkIGlPUyIsImRldmljZSI6ImlQYWQxMywxNiIsInN5c3RlbV9sb2NhbGUiOiJlbi1JTiIsImNsaWVudF92ZXJzaW9uIjoiMTI0LjAiLCJyZWxlYXNlX2NoYW5uZWwiOiJzdGFibGUiLCJkZXZpY2VfYWR2ZXJ0aXNlcl9pZCI6IjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMCIsImRldmljZV92ZW5kb3JfaWQiOiJBMTgzNkNFRC1BRDI5LTRGRTAtQjVDNC0zODQ0NDU0MEFFQTciLCJicm93c2VyX3VzZXJfYWdlbnQiOiIiLCJicm93c2VyX3ZlcnNpb24iOiIiLCJvc192ZXJzaW9uIjoiMTUuNC4xIiwiY2xpZW50X2J1aWxkX251bWJlciI6MzIyNDcsImNsaWVudF9ldmVudF9zb3VyY2UiOm51bGx9"
	} else {
		// Browser
		ua, xsuper = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:100.0) Gecko/20100101 Firefox/100.0", "eyJvcyI6Ik1hYyBPUyBYIiwiYnJvd3NlciI6IkZpcmVmb3giLCJkZXZpY2UiOiIiLCJzeXN0ZW1fbG9jYWxlIjoiZW4tVVMiLCJicm93c2VyX3VzZXJfYWdlbnQiOiJNb3ppbGxhLzUuMCAoTWFjaW50b3NoOyBJbnRlbCBNYWMgT1MgWCAxMC4xNTsgcnY6MTAwLjApIEdlY2tvLzIwMTAwMTAxIEZpcmVmb3gvMTAwLjAiLCJicm93c2VyX3ZlcnNpb24iOiIxMDAuMCIsIm9zX3ZlcnNpb24iOiIxMC4xNSIsInJlZmVycmVyIjoiaHR0cHM6Ly93d3cuZ29vZ2xlLmNvbS8iLCJyZWZlcnJpbmdfZG9tYWluIjoid3d3Lmdvb2dsZS5jb20iLCJzZWFyY2hfZW5naW5lIjoiZ29vZ2xlIiwicmVmZXJyZXJfY3VycmVudCI6IiIsInJlZmVycmluZ19kb21haW5fY3VycmVudCI6IiIsInJlbGVhc2VfY2hhbm5lbCI6InN0YWJsZSIsImNsaWVudF9idWlsZF9udW1iZXIiOjEyNzEzNSwiY2xpZW50X2V2ZW50X3NvdXJjZSI6bnVsbH0="
	}

	// Load instances
	tokens, err = utilities.ReadLines("tokens.txt")
	if err != nil {
		return cfg, instances, err
	}
	if len(tokens) == 0 {
		return cfg, instances, fmt.Errorf("no tokens found in tokens.txt")
	}

	if cfg.ProxySettings.ProxyFromFile {
		proxies, err = utilities.ReadLines("proxies.txt")
		if err != nil {
			return cfg, instances, err
		}
		if len(proxies) == 0 {
			return cfg, instances, fmt.Errorf("no proxies found in proxies.txt")
		}
	}
	var Gproxy string
	var instanceToken string
	var email string
	var password string
	reg := regexp.MustCompile(`(.+):(.+):(.+)`)
	for i := 0; i < len(tokens); i++ {
		if reg.MatchString(tokens[i]) {
			parts := strings.Split(tokens[i], ":")
			instanceToken = parts[2]
			email = parts[0]
			password = parts[1]
		} else {
			instanceToken = tokens[i]
		}
		if cfg.ProxySettings.ProxyFromFile {
			proxy = proxies[rand.Intn(len(proxies))]
			Gproxy = proxy
		} else {
			proxy = ""
		}
		client, err := InitClient(proxy, cfg)
		if err != nil {
			return cfg, instances, fmt.Errorf("couldn't initialize client: %v", err)
		}
		// proxy is put in struct only to be used by gateway. If proxy for gateway is disabled, it will be empty
		if !cfg.ProxySettings.GatewayProxy {
			Gproxy = ""
		}
		instances = append(instances, Instance{Client: client, Token: instanceToken, Proxy: proxy, Config: cfg, GatewayProxy: Gproxy, Email: email, Password: password, UserAgent: ua, XSuper: xsuper})
	}
	if len(instances) == 0 {
		color.Red("[!] You may be using 0 tokens")
	}
	var empty Config
	if cfg == empty {
		color.Red("[!] You may be using a malformed config.json")
	}
	return cfg, instances, nil

}

func SetMessages(instances []Instance, messages []Message) error {
	var err error
	if len(messages) == 0 {
		messages, err = GetMessage()
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			return fmt.Errorf("no messages found in messages.txt")
		}
		for i := 0; i < len(instances); i++ {
			instances[i].Messages = messages
		}
	} else {
		for i := 0; i < len(instances); i++ {
			instances[i].Messages = messages
		}
	}

	return nil
}

func (in *Instance) CensorToken() string {
	if len(in.Token) == 0 {
		return ""
	}
	if in.Config.OtherSettings.CensorToken {
		var censored string
		l := len(in.Token)
		uncensoredPart := int(2 * l / 3)
		for i := 0; i < l; i++ {
			if i < uncensoredPart {
				censored += string(in.Token[i])
			} else {
				censored += "*"
			}
		}
		return censored
	} else {
		return in.Token
	}

}
