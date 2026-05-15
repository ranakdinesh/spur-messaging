package meta

import (
	"fmt"
	"regexp"
	"strings"
)

type ErrorResponse struct {
	Error MetaError `json:"error"`
}

type MetaError struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode,omitempty"`
	TraceID      string `json:"fbtrace_id,omitempty"`
	RawBody      []byte `json:"-"`
}

func (e *MetaError) Error() string {
	if e == nil {
		return "meta error"
	}
	if e.ErrorSubcode != 0 {
		return fmt.Sprintf("meta error: %s (type=%s code=%d subcode=%d trace_id=%s)", e.SafeMessage(), e.Type, e.Code, e.ErrorSubcode, e.TraceID)
	}
	return fmt.Sprintf("meta error: %s (type=%s code=%d trace_id=%s)", e.SafeMessage(), e.Type, e.Code, e.TraceID)
}

func (e *MetaError) IsRateLimit() bool {
	if e == nil {
		return false
	}
	switch e.Code {
	case 4, 17, 32, 613, 80004:
		return true
	default:
		return strings.Contains(strings.ToLower(e.Message), "rate limit") ||
			strings.Contains(strings.ToLower(e.Message), "too many calls")
	}
}

func (e *MetaError) IsAuthError() bool {
	if e == nil {
		return false
	}
	if e.Code == 190 || e.Code == 102 {
		return true
	}
	msg := strings.ToLower(e.Message)
	return strings.Contains(msg, "access token") ||
		strings.Contains(msg, "oauth") ||
		strings.Contains(msg, "session has expired")
}

func (e *MetaError) IsPermissionError() bool {
	if e == nil {
		return false
	}
	switch e.Code {
	case 10, 200, 283, 299:
		return true
	default:
		msg := strings.ToLower(e.Message)
		return strings.Contains(msg, "permission") ||
			strings.Contains(msg, "permissions") ||
			strings.Contains(msg, "not authorized")
	}
}

func (e *MetaError) IsTemplateError() bool {
	if e == nil {
		return false
	}
	if e.Code >= 132000 && e.Code < 133000 {
		return true
	}
	switch e.ErrorSubcode {
	case 2388001, 2388002, 2388003, 2388004, 2388024:
		return true
	default:
		msg := strings.ToLower(e.Message)
		return strings.Contains(msg, "template") ||
			strings.Contains(msg, "message template")
	}
}

func (e *MetaError) IsTemporary() bool {
	if e == nil {
		return false
	}
	if e.IsRateLimit() {
		return true
	}
	switch e.Code {
	case 1, 2, 4, 17, 32, 613, 80004:
		return true
	default:
		msg := strings.ToLower(e.Message)
		return strings.Contains(msg, "temporarily unavailable") ||
			strings.Contains(msg, "temporary") ||
			strings.Contains(msg, "try again later") ||
			strings.Contains(msg, "service unavailable")
	}
}

func (e *MetaError) SafeMessage() string {
	if e == nil {
		return "meta error"
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "meta error"
	}
	return redactSecrets(msg)
}

type HTTPError struct {
	StatusCode int
	MetaError  *MetaError
}

func (e *HTTPError) Error() string {
	if e == nil {
		return "meta http error"
	}
	if e.MetaError != nil {
		return fmt.Sprintf("meta http error: status=%d %s", e.StatusCode, e.MetaError.Error())
	}
	return fmt.Sprintf("meta http error: status=%d", e.StatusCode)
}

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(access[_ -]?token|app[_ -]?secret|client[_ -]?secret|auth[_ -]?token|bearer)\s*[:= ]\s*['"]?[^'"\s,;]+`),
	regexp.MustCompile(`(?i)EA[A-Za-z0-9_-]{12,}`),
	regexp.MustCompile(`(?i)(token|secret)=([^&\s]+)`),
}

func redactSecrets(input string) string {
	out := input
	out = secretPatterns[0].ReplaceAllString(out, "$1=[REDACTED]")
	out = secretPatterns[1].ReplaceAllString(out, "[REDACTED]")
	out = secretPatterns[2].ReplaceAllString(out, "$1=[REDACTED]")
	return out
}
