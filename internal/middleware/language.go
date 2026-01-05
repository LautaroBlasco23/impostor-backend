package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type contextKey string

const LanguageKey contextKey = "language"

const DefaultLanguage = "en"

var supportedLanguages = map[string]bool{
	"en": true,
	"es": true,
}

func Language() fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Accept-Language")
		lang := parseAcceptLanguage(header)
		ctx := context.WithValue(c.Context(), LanguageKey, lang)
		c.SetUserContext(ctx)
		return c.Next()
	}
}

func parseAcceptLanguage(header string) string {
	if header == "" {
		return DefaultLanguage
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(strings.Split(part, ";")[0])
		if len(lang) >= 2 {
			lang = strings.ToLower(lang[:2])
			if supportedLanguages[lang] {
				return lang
			}
		}
	}
	return DefaultLanguage
}

func GetLanguage(ctx context.Context) string {
	if lang, ok := ctx.Value(LanguageKey).(string); ok {
		return lang
	}
	return DefaultLanguage
}
