package domain

import "testing"

func TestDetectConsentKeyword(t *testing.T) {
	tests := []struct {
		name string
		text string
		want ConsentKeywordAction
	}{
		{name: "english opt out", text: " STOP ", want: ConsentKeywordOptOut},
		{name: "english phrase opt out", text: "unsubscribe!", want: ConsentKeywordOptOut},
		{name: "hindi opt out", text: "रोकें", want: ConsentKeywordOptOut},
		{name: "arabic opt out", text: "إلغاء الاشتراك", want: ConsentKeywordOptOut},
		{name: "english opt in", text: "yes", want: ConsentKeywordOptIn},
		{name: "hindi opt in", text: "हाँ", want: ConsentKeywordOptIn},
		{name: "arabic opt in", text: "نعم", want: ConsentKeywordOptIn},
		{name: "normal sentence", text: "what is the price?", want: ConsentKeywordUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectConsentKeyword(tt.text); got != tt.want {
				t.Fatalf("DetectConsentKeyword(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
