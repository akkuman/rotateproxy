package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/akkuman/rotateproxy"
)

var (
	baseCfg       rotateproxy.BaseConfig
	email         string
	token         string
	rule          string
	pageCount     int
	proxy         string
	checkURL      string
	checkURLwords string
	portPattern   = regexp.MustCompile(`^\d+$`)
)

func init() {
	flag.StringVar(&baseCfg.ListenAddr, "l", ":8899", "listen address")
	flag.StringVar(&baseCfg.Username, "user", "", "authentication username")
	flag.StringVar(&baseCfg.Password, "pass", "", "authentication password")
	flag.StringVar(&email, "email", "", "email address")
	flag.StringVar(&token, "token", "", "token")
	flag.StringVar(&proxy, "proxy", "", "proxy")
	flag.StringVar(&rule, "rule", `protocol=="socks5" && "Version:5 Method:No Authentication(0x00)" && after="2022-02-01" && country="CN"`, "search rule")
	flag.StringVar(&checkURL, "check", `https://www.google.com`, "check url")
	flag.StringVar(&checkURLwords, "checkWords", `Copyright The Closure Library Authors`, "words in check url")
	flag.IntVar(&baseCfg.IPRegionFlag, "region", 0, "0: all 1: cannot bypass gfw 2: bypass gfw")
	flag.IntVar(&baseCfg.SelectStrategy, "strategy", 3, "0: random, 1: Select the one with the shortest timeout, 2: Select the two with the shortest timeout, ...")
	flag.IntVar(&pageCount, "page", 5, "the page count you want to crawl")
	flag.Parse()

	if checkURL != "https://www.google.com" && checkURLwords == "Copyright The Closure Library Authors" {
		fmt.Println("You set check url but forget to set `-checkWords`!")
		os.Exit(1)
	}

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

	// print fofa query
	rotateproxy.InfoLog(rotateproxy.Notice("You fofa query for rotateproxy is : %v", rule))

	baseCfg.ListenAddr = strings.TrimSpace(baseCfg.ListenAddr)

	if portPattern.Match([]byte(baseCfg.ListenAddr)) {
		baseCfg.ListenAddr = ":" + baseCfg.ListenAddr
	}

	rotateproxy.StartRunCrawler(token, email, rule, pageCount, proxy)
	rotateproxy.StartCheckProxyAlive(checkURL, checkURLwords)
	c := rotateproxy.NewRedirectClient(rotateproxy.WithConfig(&baseCfg))
	c.Serve()
	select {}
}
