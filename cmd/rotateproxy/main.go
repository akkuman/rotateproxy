package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

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
	flag.StringVar(&rule, "rule", fmt.Sprintf(`protocol=="socks5" && "Version:5 Method:No Authentication(0x00)" && after="%s" && country="CN"`, time.Now().AddDate(0, -3, 0).Format(time.DateOnly)), "search rule")
	flag.StringVar(&checkURL, "check", ``, "check url")
	flag.StringVar(&checkURLwords, "checkWords", ``, "words in check url")
	flag.IntVar(&baseCfg.IPRegionFlag, "region", 0, "0: all 1: cannot bypass gfw 2: bypass gfw")
	flag.IntVar(&baseCfg.SelectStrategy, "strategy", 1, "0: random, 1: Select the one with the shortest timeout")
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
	rotateproxy.InfoLog(rotateproxy.Info("You fofa query for rotateproxy is : %v", rule))
	rotateproxy.InfoLog(rotateproxy.Info("Check Proxy URL: %v", checkURL))
	rotateproxy.InfoLog(rotateproxy.Info("Check Proxy Words: %v", checkURLwords))

	baseCfg.ListenAddr = strings.TrimSpace(baseCfg.ListenAddr)

	if portPattern.Match([]byte(baseCfg.ListenAddr)) {
		baseCfg.ListenAddr = ":" + baseCfg.ListenAddr
	}

	c := make(chan os.Signal)
	// 监听信号
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()
	rotateproxy.StartRunCrawler(ctx, token, email, rule, pageCount, proxy)
	rotateproxy.StartCheckProxyAlive(ctx, checkURL, checkURLwords)
	go func() {
		c := rotateproxy.NewRedirectClient(rotateproxy.WithConfig(&baseCfg))
		c.Serve(ctx)
	}()

	<- c
	err := rotateproxy.CloseDB()
	if err != nil {
		rotateproxy.ErrorLog(rotateproxy.Warn("Error closing db: %v", err))
		os.Exit(1)
	}
}
