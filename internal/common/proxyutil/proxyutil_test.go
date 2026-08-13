package proxyutil

import (
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// socks4Server 是最小 SOCKS4/SOCKS4A 回环代理：记录握手请求并转发到目标。
type socks4Server struct {
	ln          net.Listener
	mu          sync.Mutex
	requests    []string
	targetPorts []int
}

func newSocks4Server(t *testing.T) *socks4Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &socks4Server{ln: ln}
	go server.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return server
}

func (s *socks4Server) addr() string { return s.ln.Addr().String() }

func (s *socks4Server) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *socks4Server) handle(conn net.Conn) {
	defer conn.Close()
	header := make([]byte, 8)
	if _, err := io.ReadFull(conn, header); err != nil {
		return
	}
	if header[0] != 0x04 || header[1] != 0x01 {
		return
	}
	port := int(binary.BigEndian.Uint16(header[2:4]))
	ip := header[4:8]
	// user id 直到 \x00
	userId := make([]byte, 0, 16)
	one := make([]byte, 1)
	for {
		if _, err := conn.Read(one); err != nil || one[0] == 0 {
			break
		}
		userId = append(userId, one[0])
	}
	host := ""
	if ip[0] == 0 && ip[1] == 0 && ip[2] == 0 && ip[3] == 1 {
		// SOCKS4A：域名直到 \x00
		domain := make([]byte, 0, 64)
		for {
			if _, err := conn.Read(one); err != nil || one[0] == 0 {
				break
			}
			domain = append(domain, one[0])
		}
		host = string(domain)
	} else {
		host = net.IP(ip).String()
	}
	s.mu.Lock()
	s.requests = append(s.requests, host+":"+strconv.Itoa(port))
	s.targetPorts = append(s.targetPorts, port)
	s.mu.Unlock()
	// 连接到目标并回 0x5A 成功。
	target, err := net.Dial("tcp", host+":"+strconv.Itoa(port))
	if err != nil {
		_, _ = conn.Write([]byte{0x00, 0x5B, 0, 0, 0, 0, 0, 0})
		return
	}
	defer target.Close()
	_, _ = conn.Write([]byte{0x00, 0x5A, 0, 0, 0, 0, 0, 0})
	go io.Copy(target, conn)
	io.Copy(conn, target)
}

func (s *socks4Server) sawRequest() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.requests...)
}

// TestSocks4ProxyHandshake 验证 SOCKS4 握手经回环代理完成并转发。
func TestSocks4ProxyHandshake(t *testing.T) {
	proxy := newSocks4Server(t)
	target := httpServer(t, "socks4 ok")

	transport, err := Transport("socks4://" + proxy.addr())
	if err != nil {
		t.Fatalf("Transport: %v", err)
	}
	client := &http.Client{Transport: transport}
	response, err := client.Get(target)
	if err != nil {
		t.Fatalf("GET through socks4 proxy: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if string(body) != "socks4 ok" {
		t.Fatalf("body = %q", body)
	}
	requests := proxy.sawRequest()
	if len(requests) != 1 || !strings.HasPrefix(requests[0], "127.0.0.1:") {
		t.Fatalf("proxy saw requests = %v", requests)
	}
}

// TestSocks4aProxyHandshakeWithDomain 域名目标走 SOCKS4A（0.0.0.1 + 域名）。
func TestSocks4aProxyHandshakeWithDomain(t *testing.T) {
	proxy := newSocks4Server(t)
	transport, err := Transport("socks4a://" + proxy.addr())
	if err != nil {
		t.Fatalf("Transport: %v", err)
	}
	client := &http.Client{Transport: transport}
	// 目标主机名必须可解析，否则 socks4a 代理转发失败；用 localhost。
	if _, err := client.Get("http://localhost:1"); err == nil {
		// 连接可能失败（目标拒绝），但握手必须发生。
	}
	requests := proxy.sawRequest()
	if len(requests) != 1 || !strings.HasPrefix(requests[0], "localhost:") {
		t.Fatalf("socks4a proxy saw requests = %v", requests)
	}
}

// TestSocks4RejectsDomainWithoutSocks4a 纯 SOCKS4 对域名目标必须显式失败。
func TestSocks4RejectsDomainWithoutSocks4a(t *testing.T) {
	transport, err := Transport("socks4://127.0.0.1:9")
	if err != nil {
		t.Fatalf("Transport: %v", err)
	}
	client := &http.Client{Transport: transport}
	if _, err := client.Get("http://localhost:1"); err == nil {
		t.Fatal("socks4 accepted a domain target")
	}
}

// TestTransportUnsupportedScheme 未知 scheme 显式拒绝。
func TestTransportUnsupportedScheme(t *testing.T) {
	if _, err := Transport("ftp://proxy.example:21"); err == nil {
		t.Fatal("Transport accepted ftp://")
	}
}

// TestTransportNilForEmptyProxy 空 proxy 返回 nil（回退 DefaultClient）。
func TestTransportNilForEmptyProxy(t *testing.T) {
	transport, err := Transport("")
	if err != nil || transport != nil {
		t.Fatalf("Transport(\"\") = %v, %v", transport, err)
	}
}

func httpServer(t *testing.T, body string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, body)
	})}
	go server.Serve(ln)
	t.Cleanup(func() { _ = server.Close() })
	return "http://" + ln.Addr().String()
}
