package rotateproxy

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

var (
	largeBufferSize = 32 * 1024 // 32KB large buffer
)

type BaseConfig struct {
	ListenAddr string
}

type RedirectClient struct {
	config *BaseConfig
}

type RedirectClientOption func(*RedirectClient)

func WithConfig(config *BaseConfig) RedirectClientOption {
	return func(c *RedirectClient) {
		c.config = config
	}
}

func NewRedirectClient(opts ...RedirectClientOption) *RedirectClient {
	c := &RedirectClient{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *RedirectClient) Serve() error {
	l, err := net.Listen("tcp", c.config.ListenAddr)
	if err != nil {
		return err
	}
	for IsProxyURLBlank() {
		fmt.Println("[*] waiting for crawl proxy...")
		time.Sleep(3 * time.Second)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("[!] accept error: %v\n", err)
			continue
		}
		go c.HandleConn(conn)
	}
}

func (c *RedirectClient) HandleConn(conn net.Conn) {
	key, err := RandomProxyURL()
	if err != nil {
		errConn := closeConn(conn)
		if errConn != nil {
			fmt.Printf("[!] close connect error: %v\n", errConn)
		}
		return
	}
	key = strings.TrimPrefix(key, "socks5://")
	cc, err := net.DialTimeout("tcp", key, 20*time.Second)
	if err != nil {
		fmt.Printf("[!] cannot connect to %v\n", key)
	}
	go func() {
		err = transport(conn, cc)
		if err != nil {
			fmt.Printf("[!] connect error: %v\n", err)
			errConn := closeConn(conn)
			if errConn != nil {
				fmt.Printf("[!] close connect error: %v\n", errConn)
			}
			errConn = closeConn(cc)
			if errConn != nil {
				fmt.Printf("[!] close upstream connect error: %v\n", errConn)
			}
		}
	}()
}

func closeConn(conn net.Conn) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	err = conn.Close()
	return err
}

func transport(rw1, rw2 io.ReadWriter) error {
	errc := make(chan error, 1)
	go func() {
		errc <- copyBuffer(rw1, rw2)
	}()

	go func() {
		errc <- copyBuffer(rw2, rw1)
	}()

	err := <-errc
	if err != nil && err == io.EOF {
		err = nil
	}
	return err
}

func copyBuffer(dst io.Writer, src io.Reader) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	buf := make([]byte, largeBufferSize)

	_, err = io.CopyBuffer(dst, src, buf)
	return err
}
