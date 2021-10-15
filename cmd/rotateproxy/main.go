package main

import (
	"flag"

	"github.com/akkuman/rotateproxy"
)

var (
	baseCfg rotateproxy.BaseConfig
	email   string
	token   string
	rule    string
)

func init() {
	flag.StringVar(&baseCfg.ListenAddr, "l", ":8899", "listen address")
	flag.StringVar(&email, "email", "", "email address")
	flag.StringVar(&token, "token", "", "token")
	flag.StringVar(&rule, "rule", `protocol=="socks5" && "Version:5 Method:No Authentication(0x00)" && after="2021-08-01" && country="CN"`, "search rule")
	flag.Parse()
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	if !isFlagPassed("email") || !isFlagPassed("token") {
		flag.Usage()
		return
	}

	rotateproxy.StartRunCrawler(token, email, rule)
	rotateproxy.StartCheckProxyAlive()
	c := rotateproxy.NewRedirectClient(rotateproxy.WithConfig(&baseCfg))
	c.Serve()
	select {}
}
