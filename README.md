# rotateproxy

利用fofa搜索socks5开放代理进行代理池轮切的工具

## 帮助

```shell
> .\rotateproxy.exe -h
Usage of rotateproxy.exe:
  -email string
        email address
  -l string
        listen address (default ":8899")
  -page int
        the page count you want to crawl (default 5)
  -pass string
        authentication password
  -region int
        0: all 1: cannot bypass gfw 2: bypass gfw
  -rule string
        search rule (default "protocol==\"socks5\" && \"Version:5 Method:No Authentication(0x00)\" && after=\"2021-08-01\" && country=\"CN\"")
  -token string
        token
  -user string
        authentication username
```

## 安装

```shell
go get -u github.com/akkuman/rotateproxy/cmd/...
```

或者到[发布页面](https://github.com/akkuman/rotateproxy/releases/latest)进行下载

## 效果展示

![](./pics/curl-run.jpg)