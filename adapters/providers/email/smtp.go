package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ranakdinesh/spur-messaging/core/domain"
	"github.com/ranakdinesh/spur-messaging/core/ports"
)

const defaultSMTPTimeout = 30 * time.Second

type smtpProvider struct {
	timeout time.Duration
}

func NewSMTPProvider() ports.EmailProvider {
	return &smtpProvider{timeout: defaultSMTPTimeout}
}

func (p *smtpProvider) Channel() domain.Channel {
	return domain.ChannelEmail
}

func (p *smtpProvider) Send(ctx context.Context, cfg *domain.ProviderConfig, req ports.ProviderSendRequest) (*ports.ProviderSendResult, error) {
	if req.Text == nil {
		return nil, fmt.Errorf("smtp email body is required")
	}
	emailReq := ports.EmailSendRequest{
		To:       req.Recipient,
		HTMLBody: *req.Text,
	}
	return p.SendEmail(ctx, cfg, emailReq)
}

func (p *smtpProvider) SubmitTemplate(ctx context.Context, cfg *domain.ProviderConfig, tmpl domain.Template) (string, error) {
	return "", fmt.Errorf("templates for email are handled internally, not via provider submission")
}

func (p *smtpProvider) GetTemplateStatus(ctx context.Context, cfg *domain.ProviderConfig, providerTmplID string) (domain.TemplateStatus, *string, error) {
	return domain.TemplateStatusApproved, nil, nil
}

func (p *smtpProvider) ParseWebhook(ctx context.Context, cfg *domain.ProviderConfig, headers http.Header, body []byte) ([]ports.WebhookEvent, error) {
	return nil, nil
}

func (p *smtpProvider) ValidateWebhookSignature(cfg *domain.ProviderConfig, headers http.Header, body []byte) bool {
	return false
}

func (p *smtpProvider) SendEmail(ctx context.Context, cfg *domain.ProviderConfig, req ports.EmailSendRequest) (*ports.ProviderSendResult, error) {
	if cfg == nil {
		cfg = &domain.ProviderConfig{}
	}
	creds, err := smtpCredentials(cfg)
	if err != nil {
		return nil, err
	}
	if creds.SMTPHost == "" {
		return nil, fmt.Errorf("smtp host is required")
	}
	if creds.SMTPPort == 0 {
		creds.SMTPPort = 587
	}

	req.FromEmail = firstNonEmpty(req.FromEmail, cfg.FromEmail, os.Getenv("MESSAGING_EMAIL_FROM_ADDRESS"))
	req.FromName = firstNonEmpty(req.FromName, cfg.FromName, os.Getenv("MESSAGING_EMAIL_FROM_NAME"))
	req.ReplyTo = firstNonEmpty(req.ReplyTo, cfg.ReplyToEmail)

	if err := validateSMTPRequest(req); err != nil {
		return nil, err
	}

	messageID := "<" + uuid.NewString() + "@spur-messaging>"
	raw, err := buildSMTPMessage(req, messageID)
	if err != nil {
		return nil, err
	}

	sendCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	if err := p.send(sendCtx, creds, req.FromEmail, recipients(req), raw); err != nil {
		return nil, err
	}

	return &ports.ProviderSendResult{
		ProviderMessageID: strings.Trim(messageID, "<>"),
		Status:            domain.MessageStatusSent,
		Timestamp:         time.Now(),
	}, nil
}

func (p *smtpProvider) SendBatch(ctx context.Context, cfg *domain.ProviderConfig, reqs []ports.EmailSendRequest) ([]ports.ProviderSendResult, error) {
	results := make([]ports.ProviderSendResult, 0, len(reqs))
	for _, req := range reqs {
		result, err := p.SendEmail(ctx, cfg, req)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}
	return results, nil
}

func (p *smtpProvider) send(ctx context.Context, creds domain.EmailCredentials, from string, to []string, msg []byte) error {
	addr := net.JoinHostPort(creds.SMTPHost, strconv.Itoa(creds.SMTPPort))
	tlsMode := strings.ToLower(firstNonEmpty(creds.SMTPTLSMode, "starttls"))

	dialer := &net.Dialer{Timeout: p.timeout}
	var conn net.Conn
	var err error
	if tlsMode == "tls" {
		conn, err = (&tls.Dialer{
			NetDialer: dialer,
			Config:    &tls.Config{ServerName: creds.SMTPHost, MinVersion: tls.VersionTLS12},
		}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connect smtp server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, creds.SMTPHost)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if tlsMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: creds.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("starttls smtp connection: %w", err)
			}
		}
	}

	authMode := strings.ToLower(firstNonEmpty(creds.SMTPAuth, "plain"))
	if authMode != "none" && creds.SMTPUsername != "" {
		var auth smtp.Auth
		switch authMode {
		case "login":
			auth = loginAuth{username: creds.SMTPUsername, password: creds.SMTPPassword}
		default:
			auth = smtp.PlainAuth("", creds.SMTPUsername, creds.SMTPPassword, creds.SMTPHost)
		}
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate smtp client: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("smtp recipient %s: %w", recipient, err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp message writer: %w", err)
	}
	return client.Quit()
}

func smtpCredentials(cfg *domain.ProviderConfig) (domain.EmailCredentials, error) {
	var creds domain.EmailCredentials
	if cfg != nil && len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
			return creds, fmt.Errorf("unmarshal smtp credentials: %w", err)
		}
	}
	if creds.SMTPHost == "" {
		creds.SMTPHost = os.Getenv("MESSAGING_SMTP_HOST")
	}
	if creds.SMTPPort == 0 {
		if value := os.Getenv("MESSAGING_SMTP_PORT"); value != "" {
			port, err := strconv.Atoi(value)
			if err != nil {
				return creds, fmt.Errorf("invalid MESSAGING_SMTP_PORT: %w", err)
			}
			creds.SMTPPort = port
		}
	}
	creds.SMTPUsername = firstNonEmpty(creds.SMTPUsername, os.Getenv("MESSAGING_SMTP_USERNAME"))
	creds.SMTPPassword = firstNonEmpty(creds.SMTPPassword, os.Getenv("MESSAGING_SMTP_PASSWORD"))
	creds.SMTPAuth = firstNonEmpty(creds.SMTPAuth, os.Getenv("MESSAGING_SMTP_AUTH"))
	creds.SMTPTLSMode = firstNonEmpty(creds.SMTPTLSMode, os.Getenv("MESSAGING_SMTP_TLS_MODE"))
	return creds, nil
}

func validateSMTPRequest(req ports.EmailSendRequest) error {
	if _, err := mail.ParseAddress(req.FromEmail); err != nil {
		return fmt.Errorf("invalid from email: %w", err)
	}
	if _, err := mail.ParseAddress(req.To); err != nil {
		return fmt.Errorf("invalid recipient email: %w", err)
	}
	for _, recipient := range append(append([]string{}, req.CC...), req.BCC...) {
		if _, err := mail.ParseAddress(recipient); err != nil {
			return fmt.Errorf("invalid recipient email: %w", err)
		}
	}
	if strings.TrimSpace(req.Subject) == "" {
		return fmt.Errorf("email subject is required")
	}
	if req.HTMLBody == "" && req.TextBody == "" {
		return fmt.Errorf("email body is required")
	}
	return nil
}

func buildSMTPMessage(req ports.EmailSendRequest, messageID string) ([]byte, error) {
	header := textproto.MIMEHeader{}
	header.Set("From", address(req.FromName, req.FromEmail))
	header.Set("To", req.To)
	if len(req.CC) > 0 {
		header.Set("Cc", strings.Join(req.CC, ", "))
	}
	if req.ReplyTo != "" {
		header.Set("Reply-To", req.ReplyTo)
	}
	header.Set("Subject", mime.QEncoding.Encode("utf-8", req.Subject))
	header.Set("Date", time.Now().Format(time.RFC1123Z))
	header.Set("Message-ID", messageID)
	header.Set("MIME-Version", "1.0")
	for key, value := range req.Headers {
		if isSafeHeader(key) && value != "" {
			header.Set(key, value)
		}
	}
	for key, value := range req.Metadata {
		if strings.HasPrefix(strings.ToLower(key), "x-") && isSafeHeader(key) && value != "" {
			header.Set(key, value)
		}
	}

	var body bytes.Buffer
	if len(req.Attachments) > 0 {
		boundary := "mixed-" + uuid.NewString()
		header.Set("Content-Type", `multipart/mixed; boundary="`+boundary+`"`)
		writeHeaders(&body, header)
		body.WriteString("\r\n")
		body.WriteString("--" + boundary + "\r\n")
		if err := writeAlternativePart(&body, req); err != nil {
			return nil, err
		}
		for _, attachment := range req.Attachments {
			body.WriteString("\r\n--" + boundary + "\r\n")
			writeAttachment(&body, attachment)
		}
		body.WriteString("\r\n--" + boundary + "--\r\n")
		return body.Bytes(), nil
	}

	if req.HTMLBody != "" && req.TextBody != "" {
		boundary := "alt-" + uuid.NewString()
		header.Set("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
		writeHeaders(&body, header)
		body.WriteString("\r\n")
		writeTextPart(&body, boundary, "text/plain", req.TextBody)
		writeTextPart(&body, boundary, "text/html", req.HTMLBody)
		body.WriteString("--" + boundary + "--\r\n")
		return body.Bytes(), nil
	}

	contentType := "text/plain"
	content := req.TextBody
	if req.HTMLBody != "" {
		contentType = "text/html"
		content = req.HTMLBody
	}
	header.Set("Content-Type", contentType+`; charset="UTF-8"`)
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	writeHeaders(&body, header)
	body.WriteString("\r\n")
	body.WriteString(content)
	return body.Bytes(), nil
}

func writeAlternativePart(body *bytes.Buffer, req ports.EmailSendRequest) error {
	boundary := "alt-" + uuid.NewString()
	body.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
	if req.TextBody != "" {
		writeTextPart(body, boundary, "text/plain", req.TextBody)
	}
	if req.HTMLBody != "" {
		writeTextPart(body, boundary, "text/html", req.HTMLBody)
	}
	body.WriteString("--" + boundary + "--\r\n")
	return nil
}

func writeTextPart(body *bytes.Buffer, boundary, contentType, content string) {
	body.WriteString("--" + boundary + "\r\n")
	body.WriteString("Content-Type: " + contentType + "; charset=\"UTF-8\"\r\n")
	body.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	body.WriteString(content)
	body.WriteString("\r\n")
}

func writeAttachment(body *bytes.Buffer, attachment domain.EmailAttachment) {
	contentType := firstNonEmpty(attachment.ContentType, "application/octet-stream")
	disposition := "attachment"
	if attachment.ContentID != "" {
		disposition = "inline"
	}
	body.WriteString("Content-Type: " + contentType + "\r\n")
	body.WriteString("Content-Transfer-Encoding: base64\r\n")
	body.WriteString("Content-Disposition: " + disposition + "; filename=\"" + escapeHeaderParam(attachment.Filename) + "\"\r\n")
	if attachment.ContentID != "" {
		body.WriteString("Content-ID: <" + escapeHeaderParam(attachment.ContentID) + ">\r\n")
	}
	body.WriteString("\r\n")
	body.WriteString(wrapBase64(attachment.Content))
	body.WriteString("\r\n")
}

func writeHeaders(body *bytes.Buffer, headers textproto.MIMEHeader) {
	for key, values := range headers {
		for _, value := range values {
			body.WriteString(key + ": " + value + "\r\n")
		}
	}
}

func recipients(req ports.EmailSendRequest) []string {
	all := []string{req.To}
	all = append(all, req.CC...)
	all = append(all, req.BCC...)
	return all
}

func address(name, email string) string {
	if name == "" {
		return email
	}
	return (&mail.Address{Name: name, Address: email}).String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func isSafeHeader(key string) bool {
	return key != "" && !strings.ContainsAny(key, "\r\n:")
}

func escapeHeaderParam(value string) string {
	return strings.NewReplacer("\r", "", "\n", "", `"`, "").Replace(value)
}

func wrapBase64(value string) string {
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	if len(value) <= 76 {
		return value
	}
	var b strings.Builder
	for len(value) > 76 {
		b.WriteString(value[:76])
		b.WriteString("\r\n")
		value = value[76:]
	}
	b.WriteString(value)
	return b.String()
}

type loginAuth struct {
	username string
	password string
}

func (a loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	return []byte(a.password), nil
}
