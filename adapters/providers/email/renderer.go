package email

import (
	"regexp"
	"strings"
)

// Render replaces {{variable_name}} placeholders with provided values.
func Render(template string, variables map[string]string) string {
	result := template
	for name, value := range variables {
		placeholder := "{{" + name + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// StripHTMLTags removes all HTML tags from a string to create a plain text fallback.
func StripHTMLTags(html string) string {
	// Simple regex to strip HTML tags
	re := regexp.MustCompile("<[^>]*>")
	text := re.ReplaceAllString(html, "")

	// Decode some common entities (optional, but good for quality)
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")

	return strings.TrimSpace(text)
}
