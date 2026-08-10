package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/openprx/fenfa/internal/store"
)

const (
	EncryptionNone     = "none"
	EncryptionStartTLS = "starttls"
	EncryptionSSL      = "ssl"
	DefaultSMTPPort    = 587

	defaultSMTPTimeout = 15 * time.Second
)

// SMTPConfig contains the SMTP connection and envelope settings. Passwords
// are intentionally kept out of API response types and logs.
type SMTPConfig struct {
	Host       string
	Port       int
	Encryption string
	Username   string
	Password   string
	From       string
	To         string
}

type Message struct {
	Subject string
	Body    string
}

// Sender is the seam used by handlers and tests to send one email.
type Sender interface {
	Send(ctx context.Context, config SMTPConfig, message Message) error
}

// SMTPMailer sends plain-text messages through an SMTP server.
type SMTPMailer struct {
	// TLSConfig is optional and primarily useful for tests or custom trust
	// stores. Production callers should leave it nil to use system roots.
	TLSConfig *tls.Config
}

func SMTPConfigFromSettings(settings store.SystemSettings) SMTPConfig {
	port := settings.SMTPPort
	if port == 0 {
		port = DefaultSMTPPort
	}
	encryption := strings.ToLower(strings.TrimSpace(settings.SMTPEncryption))
	if encryption == "" {
		encryption = EncryptionStartTLS
	}
	return SMTPConfig{
		Host:       strings.TrimSpace(settings.SMTPHost),
		Port:       port,
		Encryption: encryption,
		Username:   strings.TrimSpace(settings.SMTPUsername),
		Password:   settings.SMTPPassword,
		From:       strings.TrimSpace(settings.SMTPFrom),
		To:         strings.TrimSpace(settings.SMTPTo),
	}
}

func NormalizeSMTPConfig(config SMTPConfig) SMTPConfig {
	config.Host = strings.TrimSpace(config.Host)
	config.Username = strings.TrimSpace(config.Username)
	config.From = strings.TrimSpace(config.From)
	config.To = strings.TrimSpace(config.To)
	config.Encryption = strings.ToLower(strings.TrimSpace(config.Encryption))
	if config.Port == 0 {
		config.Port = DefaultSMTPPort
	}
	if config.Encryption == "" {
		config.Encryption = EncryptionStartTLS
	}
	return config
}

func ValidateSMTPConfig(config SMTPConfig) error {
	config = NormalizeSMTPConfig(config)
	if config.Host == "" || strings.ContainsAny(config.Host, "\r\n") {
		return fmt.Errorf("SMTP host is required")
	}
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("SMTP port must be between 1 and 65535")
	}
	switch config.Encryption {
	case EncryptionNone, EncryptionStartTLS, EncryptionSSL:
	default:
		return fmt.Errorf("unsupported SMTP encryption mode")
	}
	if config.Username != "" && config.Password == "" {
		return fmt.Errorf("SMTP password is required when username is set")
	}
	if config.Username == "" && config.Password != "" {
		return fmt.Errorf("SMTP username is required when password is set")
	}
	if config.Encryption == EncryptionNone && config.Username != "" {
		return fmt.Errorf("SMTP authentication requires TLS encryption")
	}
	if _, err := envelopeAddress(config.From, "sender"); err != nil {
		return err
	}
	if _, err := envelopeAddress(config.To, "recipient"); err != nil {
		return err
	}
	return nil
}

func (mailer SMTPMailer) Send(ctx context.Context, config SMTPConfig, message Message) error {
	config = NormalizeSMTPConfig(config)
	if err := ValidateSMTPConfig(config); err != nil {
		return err
	}
	if strings.TrimSpace(message.Subject) == "" || strings.ContainsAny(message.Subject, "\r\n") {
		return fmt.Errorf("email subject is required and must not contain line breaks")
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultSMTPTimeout)
		defer cancel()
	}

	from, _ := envelopeAddress(config.From, "sender")
	to, _ := envelopeAddress(config.To, "recipient")
	address := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	tlsConfig := mailer.tlsConfig(config.Host)

	var conn net.Conn
	var err error
	if config.Encryption == EncryptionSSL {
		conn, err = (&tls.Dialer{NetDialer: &net.Dialer{}, Config: tlsConfig}).DialContext(ctx, "tcp", address)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer client.Close()

	if config.Encryption == EncryptionStartTLS {
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if config.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host)); err != nil {
			return fmt.Errorf("authenticate with SMTP server: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n", from, to, message.Subject, time.Now().UTC().Format(time.RFC1123Z), message.Body); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	return nil
}

func (mailer SMTPMailer) tlsConfig(host string) *tls.Config {
	if mailer.TLSConfig != nil {
		config := mailer.TLSConfig.Clone()
		config.ServerName = host
		if config.MinVersion == 0 {
			config.MinVersion = tls.VersionTLS12
		}
		return config
	}
	return &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
}

func envelopeAddress(value, label string) (string, error) {
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("SMTP %s address is required", label)
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address == "" {
		return "", fmt.Errorf("SMTP %s address is invalid", label)
	}
	return parsed.Address, nil
}
