package identity

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"waba.local/control/internal/config"
)

type testMailbox struct {
	mu       sync.Mutex
	messages []string
}

func (m *testMailbox) all() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.messages...)
}
func localSMTP(t *testing.T, mode string) (SMTPTransport, *testMailbox) {
	t.Helper()
	cert := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(cert.Close)
	roots := x509.NewCertPool()
	roots.AddCert(cert.Certificate())
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = listener.Close() })
	box := &testMailbox{}
	go func() {
		for {
			conn, e := listener.Accept()
			if e != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				if mode == "tls" {
					c = tls.Server(c, cert.TLS)
				}
				_, _ = fmt.Fprint(c, "220 synthetic.local ESMTP\r\n")
				reader := bufio.NewReader(c)
				authed := false
				for {
					line, e := reader.ReadString('\n')
					if e != nil {
						return
					}
					parts := strings.Fields(line)
					if len(parts) == 0 {
						return
					}
					switch parts[0] {
					case "EHLO":
						_, _ = fmt.Fprint(c, "250-synthetic.local\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
					case "STARTTLS":
						_, _ = fmt.Fprint(c, "220 Ready\r\n")
						c = tls.Server(c, cert.TLS)
						reader = bufio.NewReader(c)
					case "AUTH":
						authed = len(parts) == 3 && parts[2] == base64.StdEncoding.EncodeToString([]byte("\x00synthetic\x00synthetic"))
						if authed {
							_, _ = fmt.Fprint(c, "235 Authenticated\r\n")
						} else {
							_, _ = fmt.Fprint(c, "535 Rejected\r\n")
						}
					case "MAIL", "RCPT":
						if !authed {
							_, _ = fmt.Fprint(c, "530 Authenticate\r\n")
						} else {
							_, _ = fmt.Fprint(c, "250 Accepted\r\n")
						}
					case "DATA":
						if !authed {
							return
						}
						_, _ = fmt.Fprint(c, "354 Data\r\n")
						var message strings.Builder
						for {
							part, e := reader.ReadString('\n')
							if e != nil {
								return
							}
							if part == ".\r\n" {
								break
							}
							message.WriteString(part)
						}
						box.mu.Lock()
						box.messages = append(box.messages, message.String())
						box.mu.Unlock()
						_, _ = fmt.Fprint(c, "250 Stored\r\n")
					case "QUIT":
						_, _ = fmt.Fprint(c, "221 Bye\r\n")
						return
					default:
						return
					}
				}
			}(conn)
		}
	}()
	host, raw, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(raw)
	return SMTPTransport{Config: config.SMTP{Host: host, Port: port, TLSMode: mode, Sender: "sender@example.invalid", Username: "synthetic", Password: "synthetic", Timeout: 3 * time.Second}, TLSConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}, box
}
func TestSMTPDeliveryTLSAndAuthentication(t *testing.T) {
	for _, mode := range []string{"tls", "starttls"} {
		t.Run(mode, func(t *testing.T) {
			sender, mailbox := localSMTP(t, mode)
			if e := sender.Send(context.Background(), Mail{ID: id(), Recipient: "recipient@example.invalid", Subject: "Synthetic test", Body: "synthetic marker"}); e != nil {
				t.Fatal(e)
			}
			if messages := mailbox.all(); len(messages) != 1 || !strings.Contains(messages[0], "synthetic marker") {
				t.Fatal("local mailbox did not receive")
			}
			sender.TLSConfig = nil
			if e := sender.Send(context.Background(), Mail{ID: id(), Recipient: "recipient@example.invalid", Subject: "Synthetic test", Body: "must not arrive"}); e == nil {
				t.Fatal("untrusted TLS accepted")
			}
			if len(mailbox.all()) != 1 {
				t.Fatal("untrusted delivery escaped")
			}
		})
	}
}
