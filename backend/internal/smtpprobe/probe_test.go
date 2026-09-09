package smtpprobe

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"waba.local/control/internal/config"
)

func TestTLSAuthenticationProbeNeverSends(t *testing.T) {
	certServer := httptest.NewTLSServer(http.NotFoundHandler())
	defer certServer.Close()
	roots := x509.NewCertPool()
	roots.AddCert(certServer.Certificate())
	for _, mode := range []string{"tls", "starttls"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				conn, e := listener.Accept()
				if e != nil {
					done <- e
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				if mode == "tls" {
					conn = tls.Server(conn, certServer.TLS)
				}
				fmt.Fprint(conn, "220 synthetic.example ESMTP\r\n")
				reader := bufio.NewReader(conn)
				for {
					line, e := reader.ReadString('\n')
					if e != nil {
						done <- e
						return
					}
					verb := strings.Fields(line)[0]
					switch verb {
					case "EHLO":
						fmt.Fprint(conn, "250-synthetic.example\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
					case "STARTTLS":
						fmt.Fprint(conn, "220 Ready\r\n")
						conn = tls.Server(conn, certServer.TLS)
						reader = bufio.NewReader(conn)
					case "AUTH":
						fmt.Fprint(conn, "235 Authenticated\r\n")
					case "NOOP":
						fmt.Fprint(conn, "250 OK\r\n")
					case "QUIT":
						fmt.Fprint(conn, "221 Bye\r\n")
						done <- nil
						return
					default:
						done <- fmt.Errorf("unexpected SMTP command %s", verb)
						return
					}
				}
			}()
			host, portText, _ := net.SplitHostPort(listener.Addr().String())
			port, _ := strconv.Atoi(portText)
			c := config.SMTP{Host: host, Port: port, TLSMode: mode, Username: "synthetic", Password: config.Secret("synthetic"), Timeout: 2 * time.Second}
			err = probe(context.Background(), c, &tls.Config{RootCAs: roots, ServerName: "example.com", MinVersion: tls.VersionTLS12})
			if err != nil {
				t.Fatal(err)
			}
			if err = <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestUntrustedTLSRejected(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()
	host, portText, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	port, _ := strconv.Atoi(portText)
	err := Probe(context.Background(), config.SMTP{Host: host, Port: port, TLSMode: "tls", Username: "test", Password: "synthetic", Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "TLS") {
		t.Fatal("untrusted certificate accepted")
	}
}
