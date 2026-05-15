package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://graph.facebook.com"
	defaultAPIVersion = "v23.0"
	defaultTimeout    = 15 * time.Second
)

type Client struct {
	baseURL    string
	apiVersion string
	httpClient *http.Client
}

type Option func(*Client)

func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		apiVersion: defaultAPIVersion,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.baseURL = strings.TrimRight(c.baseURL, "/")
	c.apiVersion = normalizeVersion(c.apiVersion)
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return c
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = baseURL
		}
	}
}

func WithAPIVersion(version string) Option {
	return func(c *Client) {
		if strings.TrimSpace(version) != "" {
			c.apiVersion = version
		}
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.httpClient = &http.Client{Timeout: timeout}
		}
	}
}

func (c *Client) APIVersion() string {
	return c.apiVersion
}

func (c *Client) SendTextMessage(ctx context.Context, accessToken, phoneNumberID string, req TextMessageRequest) (*SendMessageResponse, error) {
	payload := messageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               req.To,
		Type:             "text",
		Context:          req.Context,
		Text: &textObject{
			PreviewURL: req.PreviewURL,
			Body:       req.Body,
		},
	}
	var out SendMessageResponse
	if err := c.doJSON(ctx, http.MethodPost, accessToken, c.nodePath(phoneNumberID, "messages"), nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SendTemplateMessage(ctx context.Context, accessToken, phoneNumberID string, req TemplateMessageRequest) (*SendMessageResponse, error) {
	payload := messageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               req.To,
		Type:             "template",
		Context:          req.Context,
		Template: &templateObject{
			Name:       req.Name,
			Language:   templateLanguage{Code: req.Language},
			Components: req.Components,
		},
	}
	var out SendMessageResponse
	if err := c.doJSON(ctx, http.MethodPost, accessToken, c.nodePath(phoneNumberID, "messages"), nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SendMediaMessage(ctx context.Context, accessToken, phoneNumberID string, req MediaMessageRequest) (*SendMessageResponse, error) {
	media := &mediaObject{
		ID:      req.MediaID,
		Link:    req.Link,
		Caption: req.Caption,
	}
	payload := messageRequest{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               req.To,
		Type:             req.Type,
		Context:          req.Context,
	}
	switch req.Type {
	case "image":
		payload.Image = media
	case "video":
		payload.Video = media
	case "audio":
		payload.Audio = media
	case "document":
		payload.Document = media
	case "sticker":
		payload.Sticker = media
	default:
		return nil, fmt.Errorf("unsupported media message type: %s", req.Type)
	}

	var out SendMessageResponse
	if err := c.doJSON(ctx, http.MethodPost, accessToken, c.nodePath(phoneNumberID, "messages"), nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UploadMedia(ctx context.Context, accessToken, phoneNumberID, mediaType, filename string, media io.Reader) (*UploadMediaResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("messaging_product", "whatsapp"); err != nil {
		return nil, err
	}
	if mediaType != "" {
		if err := writer.WriteField("type", mediaType); err != nil {
			return nil, err
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, media); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	var out UploadMediaResponse
	headers := http.Header{"Content-Type": []string{writer.FormDataContentType()}}
	if err := c.do(ctx, http.MethodPost, accessToken, c.nodePath(phoneNumberID, "media"), nil, headers, &body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateMessageTemplate(ctx context.Context, accessToken, wabaID string, req CreateTemplateRequest) (*MessageTemplate, error) {
	var out MessageTemplate
	if err := c.doJSON(ctx, http.MethodPost, accessToken, c.nodePath(wabaID, "message_templates"), nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListTemplates(ctx context.Context, accessToken, wabaID string) (*TemplateListResponse, error) {
	query := url.Values{}
	query.Set("fields", "id,name,language,status,category,rejected_reason,quality_score,components")
	var out TemplateListResponse
	if err := c.doJSON(ctx, http.MethodGet, accessToken, c.nodePath(wabaID, "message_templates"), query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTemplate(ctx context.Context, accessToken, templateID string) (*MessageTemplate, error) {
	query := url.Values{}
	query.Set("fields", "id,name,language,status,category,rejected_reason,quality_score,components")
	var out MessageTemplate
	if err := c.doJSON(ctx, http.MethodGet, accessToken, c.nodePath(templateID), query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTemplateStatus(ctx context.Context, accessToken, templateID string) (*MessageTemplate, error) {
	return c.GetTemplate(ctx, accessToken, templateID)
}

func (c *Client) ExchangeCodeForToken(ctx context.Context, appID, appSecret, code string) (*OAuthTokenResponse, error) {
	query := url.Values{}
	query.Set("client_id", appID)
	query.Set("client_secret", appSecret)
	query.Set("code", code)
	var out OAuthTokenResponse
	if err := c.doJSON(ctx, http.MethodGet, "", "oauth/access_token", query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetWABA(ctx context.Context, accessToken, wabaID string) (*WABA, error) {
	query := url.Values{}
	query.Set("fields", "id,name,currency,timezone_id,business_verification_status")
	var out WABA
	if err := c.doJSON(ctx, http.MethodGet, accessToken, c.nodePath(wabaID), query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListPhoneNumbers(ctx context.Context, accessToken, wabaID string) (*PhoneNumberListResponse, error) {
	query := url.Values{}
	query.Set("fields", "id,display_phone_number,verified_name,quality_rating,messaging_limit_tier,code_verification_status,status,platform_type,throughput")
	var out PhoneNumberListResponse
	if err := c.doJSON(ctx, http.MethodGet, accessToken, c.nodePath(wabaID, "phone_numbers"), query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPhoneNumber(ctx context.Context, accessToken, phoneNumberID string) (*PhoneNumber, error) {
	query := url.Values{}
	query.Set("fields", "id,display_phone_number,verified_name,quality_rating,messaging_limit_tier,code_verification_status,status,platform_type,throughput")
	var out PhoneNumber
	if err := c.doJSON(ctx, http.MethodGet, accessToken, c.nodePath(phoneNumberID), query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RegisterPhoneNumber(ctx context.Context, accessToken, phoneNumberID, pin string) (*SuccessResponse, error) {
	payload := map[string]string{
		"messaging_product": "whatsapp",
		"pin":               pin,
	}
	var out SuccessResponse
	if err := c.doJSON(ctx, http.MethodPost, accessToken, c.nodePath(phoneNumberID, "register"), nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) VerifyPhoneNumberCode(ctx context.Context, accessToken, phoneNumberID, code string) (*SuccessResponse, error) {
	payload := map[string]string{"code": code}
	var out SuccessResponse
	if err := c.doJSON(ctx, http.MethodPost, accessToken, c.nodePath(phoneNumberID, "verify_code"), nil, payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) doJSON(ctx context.Context, method, accessToken, requestPath string, query url.Values, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal meta request: %w", err)
		}
		body = bytes.NewReader(buf)
	}
	headers := http.Header{"Content-Type": []string{"application/json"}}
	return c.do(ctx, method, accessToken, requestPath, query, headers, body, out)
}

func (c *Client) do(ctx context.Context, method, accessToken, requestPath string, query url.Values, headers http.Header, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.url(requestPath, query), body)
	if err != nil {
		return fmt.Errorf("create meta request: %w", err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do meta request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return decodeError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode meta response: %w", err)
	}
	return nil
}

func (c *Client) url(requestPath string, query url.Values) string {
	u, _ := url.Parse(c.baseURL)
	u.Path = path.Join(u.Path, c.apiVersion, requestPath)
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func (c *Client) nodePath(parts ...string) string {
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return path.Join(escaped...)
}

func decodeError(resp *http.Response) error {
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return &HTTPError{StatusCode: resp.StatusCode}
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		errResp.Error.RawBody = append([]byte(nil), body...)
		return &HTTPError{StatusCode: resp.StatusCode, MetaError: &errResp.Error}
	}
	return &HTTPError{StatusCode: resp.StatusCode}
}

func normalizeVersion(version string) string {
	version = strings.Trim(strings.TrimSpace(version), "/")
	if version == "" {
		return defaultAPIVersion
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}
