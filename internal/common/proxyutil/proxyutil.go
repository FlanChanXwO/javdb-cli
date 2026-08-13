// Package proxyutil 构造支持 CLI 全部既有代理 scheme 的标准库 HTTP
// transport：http/https 与 socks5/socks5h 由 net/http 原生支持，socks4 与
// socks4a 由本包实现拨号握手。仅依赖标准库，供图片读取与反搜 provider
// 复用 JavDB transport 相同的代理契约。
package proxyutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Transport 返回使用 proxyURL 的标准库 RoundTripper；空 proxy 返回 nil
// （调用方回退 http.DefaultClient）。
func Transport(proxyURL string) (http.RoundTripper, error) {
	if strings.TrimSpace(proxyURL) == "" {
		return nil, nil
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URL: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5", "socks5h":
		return &http.Transport{Proxy: http.ProxyURL(parsed)}, nil
	case "socks4", "socks4a":
		// 自定义拨号器直连代理；显式禁用环境变量代理，避免 ProxyFromEnvironment
		// 干扰 socks4 握手。
		return &http.Transport{DialContext: socks4DialContext(parsed, strings.ToLower(parsed.Scheme) == "socks4a"), Proxy: nil}, nil
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}
}

// socks4DialContext 建立经 SOCKS4/SOCKS4A 代理的 TCP 连接。
// SOCKS4 只支持 IPv4 目标；目标为域名时按 SOCKS4A 约定把 IP 段置为
// 0.0.0.1 并在用户标识后附加域名（socks4a=true 允许域名直接握手）。
func socks4DialContext(proxy *url.URL, allowDomain bool) func(ctx context.Context, network, address string) (net.Conn, error) {
	proxyAddress := proxy.Host
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" {
			return nil, fmt.Errorf("socks4 proxy only supports TCP, got %q", network)
		}
		host, portText, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("socks4 target %q: %w", address, err)
		}
		port, err := strconv.Atoi(portText)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("socks4 target %q has invalid port", address)
		}
		ip := net.ParseIP(host)
		if ip == nil && !allowDomain {
			return nil, fmt.Errorf("socks4 proxy cannot resolve domain %q without socks4a", host)
		}
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx, "tcp", proxyAddress)
		if err != nil {
			return nil, fmt.Errorf("dial socks4 proxy %q: %w", proxyAddress, err)
		}
		request := []byte{0x04, 0x01, byte(port >> 8), byte(port)}
		if ip != nil && ip.To4() != nil {
			// 8 字节 header + 空 user id + 终止符；只允许一个 0x00。
			request = append(request, ip.To4()...)
			request = append(request, 0x00)
		} else {
			// SOCKS4A：IP 段 0.0.0.1，后跟域名。
			request = append(request, 0, 0, 0, 1)
			request = append(request, 0x00)
			request = append(request, host...)
			request = append(request, 0x00)
		}
		if _, err := conn.Write(request); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("write socks4 request: %w", err)
		}
		response := make([]byte, 8)
		if _, err := readFull(conn, response); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("read socks4 response: %w", err)
		}
		if response[0] != 0x00 {
			_ = conn.Close()
			return nil, errors.New("socks4 proxy returned an invalid response header")
		}
		if response[1] != 0x5A {
			_ = conn.Close()
			return nil, fmt.Errorf("socks4 proxy rejected the connection (status %#02x)", response[1])
		}
		return conn, nil
	}
}

func readFull(conn net.Conn, buffer []byte) (int, error) {
	total := 0
	for total < len(buffer) {
		n, err := conn.Read(buffer[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
