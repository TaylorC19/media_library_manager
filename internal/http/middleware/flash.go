package middleware

import (
	"encoding/base64"
	"strings"
)

type Flash struct {
	Level   string
	Message string
}

func EncodeFlash(level, message string) string {
	payload := strings.TrimSpace(level) + "|" + strings.TrimSpace(message)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeFlash(value string) *Flash {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil
	}
	flash := &Flash{
		Level:   strings.TrimSpace(parts[0]),
		Message: strings.TrimSpace(parts[1]),
	}
	if flash.Level == "" || flash.Message == "" {
		return nil
	}
	return flash
}
