package email

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

type fakeSMTPServer struct {
	listener  net.Listener
	mode      string
	serverTLS *tls.Config
	clientTLS *tls.Config
	received  chan string
	done      chan struct{}
}

func newFakeSMTPServer(t *testing.T, mode string) *fakeSMTPServer {
	t.Helper()
	serverTLS, clientTLS := testTLSConfigs(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for fake SMTP server: %v", err)
	}
	server := &fakeSMTPServer{
		listener:  listener,
		mode:      mode,
		serverTLS: serverTLS,
		received:  make(chan string, 1),
		done:      make(chan struct{}),
	}
	if mode == EncryptionSSL {
		server.listener = tls.NewListener(listener, serverTLS)
	}
	go server.acceptLoop()
	t.Cleanup(func() {
		_ = server.listener.Close()
		<-server.done
	})
	server.clientTLS = clientTLS
	return server
}

func (server *fakeSMTPServer) acceptLoop() {
	defer close(server.done)
	conn, err := server.listener.Accept()
	if err != nil {
		return
	}
	server.serve(conn)
}

func (server *fakeSMTPServer) serve(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	write := func(value string) bool {
		if _, err := writer.WriteString(value); err != nil {
			return false
		}
		return writer.Flush() == nil
	}

	if server.mode == EncryptionSSL {
		if !write("220 fake SMTP ready\r\n") {
			return
		}
	} else if !write("220 fake SMTP ready\r\n") {
		return
	}
	tlsActive := server.mode == EncryptionSSL
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		command := strings.ToUpper(parts[0])
		switch command {
		case "EHLO", "HELO":
			if !write("250-localhost\r\n") {
				return
			}
			if server.mode == EncryptionStartTLS && !tlsActive {
				if !write("250-STARTTLS\r\n") {
					return
				}
			}
			if !tlsActive && server.mode == EncryptionStartTLS {
				if !write("250 OK\r\n") {
					return
				}
			} else {
				if !write("250-AUTH PLAIN\r\n") || !write("250 OK\r\n") {
					return
				}
			}
		case "STARTTLS":
			if !write("220 ready for TLS\r\n") {
				return
			}
			tlsConn := tls.Server(conn, server.serverTLS)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reader = bufio.NewReader(conn)
			writer = bufio.NewWriter(conn)
			tlsActive = true
		case "AUTH":
			if !write("235 2.7.0 authenticated\r\n") {
				return
			}
		case "MAIL", "RCPT":
			if !write("250 2.1.0 accepted\r\n") {
				return
			}
		case "DATA":
			if !write("354 end with <CRLF>.<CRLF>\r\n") {
				return
			}
			var data strings.Builder
			for {
				dataLine, readErr := reader.ReadString('\n')
				if readErr != nil {
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				data.WriteString(dataLine)
			}
			server.received <- data.String()
			if !write("250 2.0.0 queued\r\n") {
				return
			}
		case "QUIT":
			_ = write("221 2.0.0 bye\r\n")
			return
		default:
			if !write("250 OK\r\n") {
				return
			}
		}
	}
}

func (server *fakeSMTPServer) clientTLSConfig() *tls.Config {
	return server.clientTLS
}

func testTLSConfigs(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test TLS key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("generate test TLS serial: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create test TLS certificate: %v", err)
	}
	serverCertificate := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: privateKey}
	rootPool := x509.NewCertPool()
	root, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse test TLS certificate: %v", err)
	}
	rootPool.AddCert(root)
	return &tls.Config{Certificates: []tls.Certificate{serverCertificate}, MinVersion: tls.VersionTLS12}, &tls.Config{RootCAs: rootPool, MinVersion: tls.VersionTLS12}
}

func (server *fakeSMTPServer) addressConfig(encryption string) SMTPConfig {
	host, portText, err := net.SplitHostPort(server.listener.Addr().String())
	if err != nil {
		panic(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		panic(err)
	}
	return SMTPConfig{
		Host:       host,
		Port:       port,
		Encryption: encryption,
		From:       "sender@example.com",
		To:         "admin@example.com",
	}
}

func TestSMTPMailerSendsThroughSupportedModes(t *testing.T) {
	for _, test := range []struct {
		name       string
		mode       string
		encryption string
		auth       bool
	}{
		{name: "none", mode: EncryptionNone, encryption: EncryptionNone},
		{name: "starttls", mode: EncryptionStartTLS, encryption: EncryptionStartTLS, auth: true},
		{name: "ssl", mode: EncryptionSSL, encryption: EncryptionSSL},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := newFakeSMTPServer(t, test.mode)
			config := server.addressConfig(test.encryption)
			mailer := SMTPMailer{TLSConfig: server.clientTLSConfig()}
			if test.auth {
				config.Username = "smtp-user"
				config.Password = "smtp-password"
			}
			err := mailer.Send(context.Background(), config, Message{Subject: "Test subject", Body: "Feedback body"})
			if err != nil {
				t.Fatalf("send email: %v", err)
			}
			select {
			case message := <-server.received:
				for _, expected := range []string{"From: sender@example.com", "To: admin@example.com", "Subject: Test subject", "Feedback body"} {
					if !strings.Contains(message, expected) {
						t.Fatalf("expected message to contain %q, got %q", expected, message)
					}
				}
			case <-time.After(time.Second):
				t.Fatalf("timed out waiting for fake SMTP message")
			}
		})
	}
}

func TestValidateSMTPConfig(t *testing.T) {
	base := SMTPConfig{Host: "smtp.example.com", Port: 587, Encryption: EncryptionStartTLS, From: "sender@example.com", To: "admin@example.com"}
	for _, test := range []struct {
		name   string
		config SMTPConfig
	}{
		{name: "missing host", config: SMTPConfig{Port: 587, Encryption: EncryptionStartTLS, From: base.From, To: base.To}},
		{name: "invalid port", config: SMTPConfig{Host: base.Host, Port: 70000, Encryption: EncryptionStartTLS, From: base.From, To: base.To}},
		{name: "invalid encryption", config: SMTPConfig{Host: base.Host, Port: 587, Encryption: "tls", From: base.From, To: base.To}},
		{name: "invalid recipient", config: SMTPConfig{Host: base.Host, Port: 587, Encryption: EncryptionStartTLS, From: base.From, To: "not-an-email"}},
		{name: "credentials without TLS", config: SMTPConfig{Host: base.Host, Port: 25, Encryption: EncryptionNone, Username: "user", Password: "pass", From: base.From, To: base.To}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateSMTPConfig(test.config); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}
