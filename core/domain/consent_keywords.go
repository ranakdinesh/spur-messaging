package domain

import (
	"strings"
	"unicode"
)

var optOutKeywords = map[string]struct{}{
	"stop":           {},
	"unsubscribe":    {},
	"cancel":         {},
	"end":            {},
	"quit":           {},
	"opt out":        {},
	"opt-out":        {},
	"no":             {},
	"बंद":            {},
	"रोक":            {},
	"रोकें":          {},
	"बंद करो":        {},
	"सदस्यता रद्द":   {},
	"توقف":           {},
	"الغاء":          {},
	"إلغاء":          {},
	"إلغاء الاشتراك": {},
	"الغاء الاشتراك": {},
	"لا":             {},
}

var optInKeywords = map[string]struct{}{
	"start":     {},
	"subscribe": {},
	"yes":       {},
	"y":         {},
	"opt in":    {},
	"opt-in":    {},
	"हाँ":       {},
	"हा":        {},
	"शुरू":      {},
	"نعم":       {},
	"اشترك":     {},
	"ابدأ":      {},
}

func DetectConsentKeyword(text string) ConsentKeywordAction {
	normalized := normalizeConsentKeyword(text)
	if normalized == "" {
		return ConsentKeywordUnknown
	}
	if _, ok := optOutKeywords[normalized]; ok {
		return ConsentKeywordOptOut
	}
	if _, ok := optInKeywords[normalized]; ok {
		return ConsentKeywordOptIn
	}
	return ConsentKeywordUnknown
}

func normalizeConsentKeyword(text string) string {
	text = strings.TrimSpace(strings.ToLower(text))
	text = strings.TrimFunc(text, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSymbol(r) || unicode.IsSpace(r)
	})
	text = strings.Join(strings.Fields(text), " ")
	return text
}
