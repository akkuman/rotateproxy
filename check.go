package rotateproxy

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type IPInfo struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
}

func CheckProxyAlive(proxyURL string) (respBody string, timeout int64, avail bool) {
	proxy, _ := url.Parse(proxyURL)
	httpclient := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(proxy),
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		},
		// shorter timeout for better proxies
		Timeout: 5 * time.Second,
	}
	startTime := time.Now()
	resp, err := httpclient.Get("https://whois.pconline.com.cn/ipJson.jsp?json=true&ip=")
	if err != nil {
		return "", 0, false
	}
	defer resp.Body.Close()
	timeout = int64(time.Since(startTime))
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, false
	}
	if resp.StatusCode != 200 {
		return "", 0, false
	}
	return string(body), timeout, true
}

func CheckProxyWithCheckURL(proxyURL string, checkURL string, checkURLwords string) (timeout int64, avail bool) {
	// InfoLog(Notice("check %s： %s", proxyURL, checkURL))
	proxy, _ := url.Parse(proxyURL)
	httpclient := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(proxy),
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		},
		Timeout: 20 * time.Second,
	}
	startTime := time.Now()
	resp, err := httpclient.Get(checkURL)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	timeout = int64(time.Since(startTime))
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return 0, false
	}

	// TODO: support regex
	if resp.StatusCode != 200 {
		return 0, false
	}

	if !strings.Contains(string(body), checkURLwords) {
		return 0, false
	}

	return timeout, true
}

func StartCheckProxyAlive(ctx context.Context, checkURL string, checkURLwords string) {
	go func() {
		ticker := time.NewTicker(120 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-crawlDone:
				InfoLog(Noticeln("Checkings"))
				checkAlive(checkURL, checkURLwords)
				InfoLog(Noticeln("Check done"))
			case <-ticker.C:
				checkAlive(checkURL, checkURLwords)
			case <- ctx.Done():
				return
			}
		}
	}()
}

func checkAlive(checkURL string, checkURLwords string) {
	proxies, err := QueryProxyURL()
	if err != nil {
		ErrorLog(Warn("[!] query db error: %v", err))
	}
	for i := range proxies {
		proxy := proxies[i]
		go func() {
			respBody, timeout, avail := CheckProxyAlive(proxy.URL)
			if avail {
				if checkURL != "" {
					timeout, avail = CheckProxyWithCheckURL(proxy.URL, checkURL, checkURLwords)
				}
				if avail {
					InfoLog(Notice("%v 可用", proxy.URL))
					SetProxyURLAvail(proxy.URL, timeout, CanBypassGFW(respBody))
					return
				}
			}
			AddProxyURLRetry(proxy.URL)
		}()
	}
}

func CanBypassGFW(respBody string) bool {
	return gjson.Get(respBody, "pro").String() == ""
}
