package meta

type SendMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID            string `json:"id"`
		MessageStatus string `json:"message_status,omitempty"`
	} `json:"messages"`
}

type TextMessageRequest struct {
	To         string          `json:"to"`
	Body       string          `json:"body"`
	PreviewURL bool            `json:"preview_url,omitempty"`
	Context    *MessageContext `json:"context,omitempty"`
}

type TemplateMessageRequest struct {
	To         string              `json:"to"`
	Name       string              `json:"name"`
	Language   string              `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
	Context    *MessageContext     `json:"context,omitempty"`
}

type MediaMessageRequest struct {
	To      string          `json:"to"`
	Type    string          `json:"type"`
	MediaID string          `json:"media_id,omitempty"`
	Link    string          `json:"link,omitempty"`
	Caption string          `json:"caption,omitempty"`
	Context *MessageContext `json:"context,omitempty"`
}

type messageRequest struct {
	MessagingProduct string              `json:"messaging_product"`
	RecipientType    string              `json:"recipient_type,omitempty"`
	To               string              `json:"to"`
	Type             string              `json:"type"`
	Context          *MessageContext     `json:"context,omitempty"`
	Text             *textObject         `json:"text,omitempty"`
	Template         *templateObject     `json:"template,omitempty"`
	Image            *mediaObject        `json:"image,omitempty"`
	Video            *mediaObject        `json:"video,omitempty"`
	Audio            *mediaObject        `json:"audio,omitempty"`
	Document         *mediaObject        `json:"document,omitempty"`
	Sticker          *mediaObject        `json:"sticker,omitempty"`
	Components       []TemplateComponent `json:"-"`
}

type MessageContext struct {
	MessageID string `json:"message_id"`
}

type textObject struct {
	PreviewURL bool   `json:"preview_url,omitempty"`
	Body       string `json:"body"`
}

type templateObject struct {
	Name       string              `json:"name"`
	Language   templateLanguage    `json:"language"`
	Components []TemplateComponent `json:"components,omitempty"`
}

type templateLanguage struct {
	Code string `json:"code"`
}

type mediaObject struct {
	ID      string `json:"id,omitempty"`
	Link    string `json:"link,omitempty"`
	Caption string `json:"caption,omitempty"`
}

type TemplateComponent struct {
	Type       string              `json:"type"`
	SubType    string              `json:"sub_type,omitempty"`
	Index      string              `json:"index,omitempty"`
	Parameters []TemplateParameter `json:"parameters,omitempty"`
}

type TemplateParameter struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	Currency *CurrencyParam `json:"currency,omitempty"`
	DateTime *DateTimeParam `json:"date_time,omitempty"`
	Image    *MediaParam    `json:"image,omitempty"`
	Video    *MediaParam    `json:"video,omitempty"`
	Document *DocumentParam `json:"document,omitempty"`
	Payload  string         `json:"payload,omitempty"`
}

type CurrencyParam struct {
	FallbackValue string `json:"fallback_value"`
	Code          string `json:"code"`
	Amount1000    int64  `json:"amount_1000"`
}

type DateTimeParam struct {
	FallbackValue string `json:"fallback_value"`
}

type MediaParam struct {
	ID   string `json:"id,omitempty"`
	Link string `json:"link,omitempty"`
}

type DocumentParam struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type UploadMediaResponse struct {
	ID string `json:"id"`
}

type CreateTemplateRequest struct {
	Name       string                     `json:"name"`
	Language   string                     `json:"language"`
	Category   string                     `json:"category"`
	Components []MessageTemplateComponent `json:"components"`
}

type MessageTemplateComponent struct {
	Type    string           `json:"type"`
	Format  string           `json:"format,omitempty"`
	Text    string           `json:"text,omitempty"`
	Example *TemplateExample `json:"example,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
}

type TemplateExample struct {
	HeaderHandle []string   `json:"header_handle,omitempty"`
	BodyText     [][]string `json:"body_text,omitempty"`
}

type TemplateButton struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	URL         string `json:"url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Example     string `json:"example,omitempty"`
}

type MessageTemplate struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name"`
	Language       string                     `json:"language"`
	Status         string                     `json:"status"`
	Category       string                     `json:"category"`
	RejectedReason string                     `json:"rejected_reason,omitempty"`
	QualityScore   *TemplateQualityScore      `json:"quality_score,omitempty"`
	Components     []MessageTemplateComponent `json:"components,omitempty"`
}

type TemplateQualityScore struct {
	Score string `json:"score,omitempty"`
}

type TemplateListResponse struct {
	Data   []MessageTemplate `json:"data"`
	Paging *Paging           `json:"paging,omitempty"`
}

type WABA struct {
	ID                         string `json:"id"`
	Name                       string `json:"name"`
	Currency                   string `json:"currency,omitempty"`
	TimezoneID                 string `json:"timezone_id,omitempty"`
	BusinessVerificationStatus string `json:"business_verification_status,omitempty"`
}

type PhoneNumber struct {
	ID                     string      `json:"id"`
	DisplayPhoneNumber     string      `json:"display_phone_number"`
	VerifiedName           string      `json:"verified_name,omitempty"`
	QualityRating          string      `json:"quality_rating,omitempty"`
	MessagingLimitTier     string      `json:"messaging_limit_tier,omitempty"`
	CodeVerificationStatus string      `json:"code_verification_status,omitempty"`
	Status                 string      `json:"status,omitempty"`
	PlatformType           string      `json:"platform_type,omitempty"`
	Throughput             *Throughput `json:"throughput,omitempty"`
}

type Throughput struct {
	Level string `json:"level,omitempty"`
}

type PhoneNumberListResponse struct {
	Data   []PhoneNumber `json:"data"`
	Paging *Paging       `json:"paging,omitempty"`
}

type Paging struct {
	Cursors *struct {
		Before string `json:"before,omitempty"`
		After  string `json:"after,omitempty"`
	} `json:"cursors,omitempty"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
}

type SuccessResponse struct {
	Success any `json:"success"`
}

type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
}

func (r SuccessResponse) OK() bool {
	switch v := r.Success.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}
