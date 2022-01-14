package rotateproxy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var crawlDone = make(chan struct{})

type fofaAPIResponse struct {
	Err     bool       `json:"error"`
	Mode    string     `json:"mode"`
	Page    int        `json:"page"`
	Query   string     `json:"query"`
	Results [][]string `json:"results"`
	Size    int        `json:"size"`
}

func addProxyURL(url string) {
	CreateProxyURL(url)
}

func RunCrawler(fofaApiKey, fofaEmail, rule string, pageNum int) (err error) {
	req, err := http.NewRequest("GET", "https://fofa.so/api/v1/search/all", nil)
	if err != nil {
		return err
	}
	rule = base64.StdEncoding.EncodeToString([]byte(rule))
	q := req.URL.Query()
	q.Add("email", fofaEmail)
	q.Add("key", fofaApiKey)
	q.Add("qbase64", rule)
	q.Add("size", "100")
	q.Add("page", fmt.Sprintf("%d", pageNum))
	q.Add("fields", "host,title,ip,domain,port,country,city,server,protocol")
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	fmt.Printf("start to parse proxy url from response\n")
	defer resp.Body.Close()
	var res fofaAPIResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return err
	}
	fmt.Printf("get %d host\n", len(res.Results))
	for _, value := range res.Results {
		host := value[0]
		addProxyURL(fmt.Sprintf("socks5://%s", host))
	}
	crawlDone <- struct{}{}
	return
}

func StartRunCrawler(fofaApiKey, fofaEmail, rule string, pageCount int) {
	runCrawlerFunc := func() {
		for i := 1; i <= 3; i++ {
			err := RunCrawler(fofaApiKey, fofaEmail, rule, i)
			if err != nil {
				fmt.Printf("[!] error: %v\n", err)
			}
		}
	}
	go func() {
		runCrawlerFunc()
		ticker := time.NewTicker(600 * time.Second)
		for range ticker.C {
			runCrawlerFunc()
		}
	}()
}
