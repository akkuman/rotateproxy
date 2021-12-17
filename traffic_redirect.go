package rotateproxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const (
	socksVer5 = 5
)

var (
	largeBufferSize = 32 * 1024 // 32KB large buffer
	ErrNotSocks5Proxy = errors.New("this is not a socks proxy server")
)

type BaseConfig struct {
	ListenAddr   string
	IPRegionFlag int // 0: all 1: cannot bypass gfw 2: bypass gfw
	Username string
	Password string
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

// handshakeWithDownstream auth for socks5 server(local)
func (c *RedirectClient) handshakeWithDownstream(conn net.Conn) (err error) {
	buf := make([]byte, 256)
	// read VER and NMETHODS
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return errors.New("reading header: " + err.Error())
	}
	ver, nMethods := int(buf[0]), int(buf[1])
	if ver != socksVer5 {
		return errors.New("invalid version")
	}
	if _, err = io.ReadFull(conn, buf[:nMethods]); err != nil {
		return errors.New("reading methods: " + err.Error())
	}
	/*
	   X'00' NO AUTHENTICATION REQUIRED
	   X'01' GSSAPI
	   X'02' USERNAME/PASSWORD
	   X'03' to X'7F' IANA ASSIGNED
	   X'80' to X'FE' RESERVED FOR PRIVATE METHODS
	   X'FF' NO ACCEPTABLE METHODS
	*/
	if !bytes.Contains(buf[:nMethods], []byte{0x02}) {
		_, err := conn.Write([]byte{socksVer5, 0xff})
		if err != nil {
			return errors.New("write need auth error: " + err.Error())
		}
		err = errors.New("method forbidden")
		return err
	}
	// USERNAME/PASSWORD
	_, err = conn.Write([]byte{socksVer5, 0x02})
	if err != nil {
		return
	}
	_, err = conn.Read(buf[0:])
	if err != nil {
		return
	}
	b0 := buf[0]
	nameLens := int(buf[1])
	uName := string(buf[2 : 2+nameLens])

	passLens := int(buf[2+nameLens])
	uPass := string(buf[2+nameLens+1 : 2+nameLens+1+passLens])

	if uName != c.config.Username || uPass != c.config.Password {
		_, _ = conn.Write([]byte{b0, 0xff})
		err = errors.New("authentication failed")
		return
	}
	// send confirmation: version 5, no authentication required
	_, err = conn.Write([]byte{b0, 0x00})
	return
}

// handshakeWithUpstream no auth for remote socks5 serer
func (c *RedirectClient) handshakeWithUpstream(conn net.Conn) (err error) {
	_, err = conn.Write([]byte{socksVer5, 0x02, 0x00, 0x01})
	if err != nil {
		return errors.New("write upstream connection failed: " + err.Error())
	}
	buf := make([]byte, 256)
	_, err = io.ReadFull(conn, buf[:2])
	if err != nil {
		return err
	}
	if buf[0] != socksVer5 && int(buf[1]) != 0x00 {
		return ErrNotSocks5Proxy
	}
	return
}

// getValidSocks5Connection 获取可用的socks5连接并完成握手阶段
func (c *RedirectClient) getValidSocks5Connection() (net.Conn, error) {
	var cc net.Conn
	for {
		key, err := RandomProxyURL(c.config.IPRegionFlag)
		if err != nil {
			return nil, err
		}
		key = strings.TrimPrefix(key, "socks5://")
		cc, err = net.DialTimeout("tcp", key, 20*time.Second)
		if err != nil {
			closeConn(cc)
			fmt.Printf("[!] cannot connect to %v\n", key)
		}
		fmt.Printf("use %v\n", key)
		// write header for remote socks5 server
		err = c.handshakeWithUpstream(cc)
		if err != nil {
			closeConn(cc)
			if errors.Is(err, ErrNotSocks5Proxy) {
				fmt.Println(err)
				continue
			}
			fmt.Printf("socks handshake with downstream failed: %v\n", err)
			continue
		}
		break
	}
	return cc, nil
}

func (c *RedirectClient) HandleConn(conn net.Conn) {
	defer closeConn(conn)
	// auth for local socks5 serer
	err := c.handshakeWithDownstream(conn)
	if err != nil {
		fmt.Printf("socks handshake with downstream failed: %v\n", err)
		return
	}
	cc, err := c.getValidSocks5Connection()
	if err != nil {
		fmt.Printf("getValidSocks5Connection failed: %v\n", err)
		return
	}
	defer closeConn(cc)
	err = transport(conn, cc)
	if err != nil {
		fmt.Printf("[!] transport error: %v\n", err)
	}
}

func closeConn(conn net.Conn) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
			fmt.Printf("close connection: %v\n", err)
		}
	}()
	err = conn.Close()
	return err
}

func transport(rw1, rw2 io.ReadWriter) error {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error{
		return copyBuffer(rw1, rw2)
	})

	g.Go(func() error{
		return copyBuffer(rw2, rw1)
	})
	var err error
	if err = g.Wait(); err != nil && err == io.EOF {
		err = nil
	}
	return err
}

func copyBuffer(dst io.Writer, src io.Reader) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("copyBuffer: %v", e)
		}
	}()
	buf := make([]byte, largeBufferSize)

	_, err = CopyBufferWithCloseErr(dst, src, buf)
	return err
}
