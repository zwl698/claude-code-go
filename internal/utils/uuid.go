package utils

import (
	"crypto/rand"
	"regexp"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidateUUID checks if a string is a valid UUID.
func ValidateUUID(maybeUUID string) bool {
	return uuidRegex.MatchString(maybeUUID)
}

// GenerateUUID generates a new UUID v4.
func GenerateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return formatUUID(b)
}

// GenerateShortID generates a short random ID (16 hex chars).
func GenerateShortID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return formatHex(b)
}

// GeneratePrefixedID generates an ID with a prefix.
func GeneratePrefixedID(prefix string) string {
	return prefix + GenerateShortID()
}

func formatUUID(b []byte) string {
	return formatHex(b[0:4]) + "-" +
		formatHex(b[4:6]) + "-" +
		formatHex(b[6:8]) + "-" +
		formatHex(b[8:10]) + "-" +
		formatHex(b[10:16])
}

func formatHex(b []byte) string {
	const hexDigits = "0123456789abcdef"
	result := make([]byte, len(b)*2)
	for i, v := range b {
		result[i*2] = hexDigits[v>>4]
		result[i*2+1] = hexDigits[v&0x0f]
	}
	return string(result)
}
