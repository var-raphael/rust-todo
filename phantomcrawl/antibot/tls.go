package antibot

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

func NewTLSClient() *http.Client {
	return newTLSClientWithProxy(nil)
}

func NewTLSClientWithProxy(proxy *url.URL) *http.Client {
	return newTLSClientWithProxy(proxy)
}

func newTLSClientWithProxy(proxy *url.URL) *http.Client {
	dialer := func(network, addr string) (net.Conn, error) {
		return dialUTLSViaProxy(network, addr, proxy)
	}

	h2transport := &http2.Transport{
		TLSClientConfig: &tls.Config{},
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return dialUTLSViaProxy(network, addr, proxy)
		},
	}

	h1transport := &http.Transport{
		DialTLS:             dialer,
		DisableCompression:  false,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &smartTransport{h2: h2transport, h1: h1transport},
	}
}

type smartTransport struct {
	h2 *http2.Transport
	h1 *http.Transport
}

func (t *smartTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.h2.RoundTrip(req)
	if err == nil {
		return resp, nil
	}
	return t.h1.RoundTrip(req)
}

func dialUTLS(network, addr string) (net.Conn, error) {
	return dialUTLSViaProxy(network, addr, nil)
}

func dialUTLSViaProxy(network, addr string, proxy *url.URL) (net.Conn, error) {
	var conn net.Conn
	var err error

	if proxy != nil {
		// Connect to proxy first, then CONNECT tunnel to target
		conn, err = net.DialTimeout("tcp", proxy.Host, 10*time.Second)
		if err != nil {
			return nil, fmt.Errorf("proxy dial failed: %w", err)
		}

		// Send HTTP CONNECT to tunnel through proxy
		fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
		if proxy.User != nil {
			user := proxy.User.Username()
			pass, _ := proxy.User.Password()
			auth := basicAuth(user, pass)
			fmt.Fprintf(conn, "Proxy-Authorization: Basic %s\r\n", auth)
		}
		fmt.Fprintf(conn, "\r\n")

		// Read CONNECT response
		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("proxy connect read failed: %w", err)
		}
		response := string(buf[:n])
		if len(response) < 12 || response[9:12] != "200" {
			conn.Close()
			return nil, fmt.Errorf("proxy CONNECT failed: %s", response)
		}
	} else {
		conn, err = net.Dial(network, addr)
		if err != nil {
			return nil, err
		}
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	uconn := utls.UClient(conn, &utls.Config{
		ServerName: host,
		NextProtos: []string{"h2", "http/1.1"},
	}, utls.HelloChrome_120)

	if err := uconn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	return uconn, nil
}

func basicAuth(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}
