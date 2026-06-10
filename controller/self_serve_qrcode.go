package controller

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"regexp"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

var legacyQRCodeImageValuePattern = regexp.MustCompile(`(?i)^https?://.+\.(png|jpe?g|webp|gif)(\?.*)?(#.*)?$`)

func isLegacyQRCodeImageValue(value string) bool {
	text := strings.TrimSpace(value)
	return strings.HasPrefix(text, "data:image/") || legacyQRCodeImageValuePattern.MatchString(text)
}

func selfServeQRCodeDisplayValue(value string) string {
	text := strings.TrimSpace(value)
	if text == "" || isLegacyQRCodeImageValue(text) {
		return text
	}

	code, err := qr.Encode(text, qr.M, qr.Auto)
	if err != nil {
		return text
	}
	code, err = barcode.Scale(code, 220, 220)
	if err != nil {
		return text
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, code); err != nil {
		return text
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}
