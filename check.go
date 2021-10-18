package rotateproxy

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
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

func CheckProxyAlive(proxyURL string) bool {
	proxy, _ := url.Parse(proxyURL)
	httpclient := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxy),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 20 * time.Second,
	}
	resp, err := httpclient.Get("https://www.baidu.com/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	if !strings.Contains(string(body), "www.baidu.com") {
		return false
	}
	return true
}

func StartCheckProxyAlive() {
	go func() {
		ticker := time.NewTicker(120 * time.Second)
		for {
			select {
			case <-crawlDone:
				fmt.Println("Checking")
				checkAlive()
				fmt.Println("Check done")
			case <-ticker.C:
				checkAlive()
			}
		}
	}()
}

func checkAlive() {
	proxies, err := QueryProxyURL()
	if err != nil {
		fmt.Printf("[!] query db error: %v\n", err)
	}
	for i := range proxies {
		proxy := proxies[i]
		go func() {
			if CheckProxyAlive(proxy.URL) {
				fmt.Printf("%v 可用\n", proxy.URL)
				SetProxyURLAvail(proxy.URL)
			} else {
				AddProxyURLRetry(proxy.URL)
			}
		}()
	}
}
