package main

import (
	"flag"
	"github.com/akkuman/rotateproxy"
	"regexp"
	"strings"
)

var (
	baseCfg     rotateproxy.BaseConfig
	email       string
	token       string
	rule        string
	pageCount   int
	portPattern = regexp.MustCompile(`^\d+$`)
)

func init() {
	flag.StringVar(&baseCfg.ListenAddr, "l", ":8899", "listen address")
	flag.StringVar(&baseCfg.Username, "user", "", "authentication username")
	flag.StringVar(&baseCfg.Password, "pass", "", "authentication password")
	flag.StringVar(&email, "email", "", "email address")
	flag.StringVar(&token, "token", "", "token")
	flag.StringVar(&rule, "rule", `protocol=="socks5" && "Version:5 Method:No Authentication(0x00)" && after="2021-08-01" && country="CN"`, "search rule")
	flag.IntVar(&baseCfg.IPRegionFlag, "region", 0, "0: all 1: cannot bypass gfw 2: bypass gfw")
	flag.IntVar(&pageCount, "page", 5, "the page count you want to crawl")
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

	baseCfg.ListenAddr = strings.TrimSpace(baseCfg.ListenAddr)

	if portPattern.Match([]byte(baseCfg.ListenAddr)) {
		baseCfg.ListenAddr = ":" + baseCfg.ListenAddr
	}

	rotateproxy.StartRunCrawler(token, email, rule, pageCount)
	rotateproxy.StartCheckProxyAlive()
	c := rotateproxy.NewRedirectClient(rotateproxy.WithConfig(&baseCfg))
	c.Serve()
	select {}
}
