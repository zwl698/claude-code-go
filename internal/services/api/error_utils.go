package api

import (
	"regexp"
	"strings"
)

// SSL error codes from OpenSSL
// See: https://www.openssl.org/docs/man3.1/man3/X509_STORE_CTX_get_error.html
var sslErrorCodes = map[string]bool{
	// Certificate verification errors
	"UNABLE_TO_VERIFY_LEAF_SIGNATURE":   true,
	"UNABLE_TO_GET_ISSUER_CERT":         true,
	"UNABLE_TO_GET_ISSUER_CERT_LOCALLY": true,
	"CERT_SIGNATURE_FAILURE":            true,
	"CERT_NOT_YET_VALID":                true,
	"CERT_HAS_EXPIRED":                  true,
	"CERT_REVOKED":                      true,
	"CERT_REJECTED":                     true,
	"CERT_UNTRUSTED":                    true,
	// Self-signed certificate errors
	"DEPTH_ZERO_SELF_SIGNED_CERT": true,
	"SELF_SIGNED_CERT_IN_CHAIN":   true,
	// Chain errors
	"CERT_CHAIN_TOO_LONG":  true,
	"PATH_LENGTH_EXCEEDED": true,
	// Hostname/altname errors
	"ERR_TLS_CERT_ALTNAME_INVALID": true,
	"HOSTNAME_MISMATCH":            true,
	// TLS handshake errors
	"ERR_TLS_HANDSHAKE_TIMEOUT":                   true,
	"ERR_SSL_WRONG_VERSION_NUMBER":                true,
	"ERR_SSL_DECRYPTION_FAILED_OR_BAD_RECORD_MAC": true,
}

// ConnectionErrorDetails represents details extracted from a connection error
type ConnectionErrorDetails struct {
	Code       string
	Message    string
	IsSSLError bool
}

// ExtractConnectionErrorDetails extracts connection error details from the error cause chain.
// The Anthropic SDK wraps underlying errors in the cause property.
// This function walks the cause chain to find the root error code/message.
func ExtractConnectionErrorDetails(err error) *ConnectionErrorDetails {
	if err == nil {
		return nil
	}

	// Walk the cause chain to find the root error with a code
	current := err
	maxDepth := 5 // Prevent infinite loops
	depth := 0

	for current != nil && depth < maxDepth {
		// Check if the error has a code field
		if coder, ok := current.(interface{ Code() string }); ok {
			code := coder.Code()
			isSSLError := sslErrorCodes[code]
			return &ConnectionErrorDetails{
				Code:       code,
				Message:    current.Error(),
				IsSSLError: isSSLError,
			}
		}

		// Check for unwrappable errors
		if unwrapper, ok := current.(interface{ Unwrap() error }); ok {
			next := unwrapper.Unwrap()
			if next == current {
				break
			}
			current = next
			depth++
		} else {
			break
		}
	}

	return nil
}

// GetSSLErrorHint returns an actionable hint for SSL/TLS errors, intended for contexts outside
// the main API client (OAuth token exchange, preflight connectivity checks)
// where FormatAPIError doesn't apply.
//
// Motivation: enterprise users behind TLS-intercepting proxies (Zscaler et al.)
// see OAuth complete in-browser but the CLI's token exchange silently fails
// with a raw SSL code. Surfacing the likely fix saves a support round-trip.
func GetSSLErrorHint(err error) string {
	details := ExtractConnectionErrorDetails(err)
	if details == nil || !details.IsSSLError {
		return ""
	}
	return "SSL certificate error (" + details.Code + "). If you are behind a corporate proxy or TLS-intercepting firewall, set NODE_EXTRA_CA_CERTS to your CA bundle path, or ask IT to allowlist *.anthropic.com. Run /doctor for details."
}

// sanitizeMessageHTML strips HTML content (e.g., CloudFlare error pages) from a message string,
// returning a user-friendly title or empty string if HTML is detected.
// Returns the original message unchanged if no HTML is found.
func sanitizeMessageHTML(message string) string {
	if strings.Contains(message, "<!DOCTYPE html") || strings.Contains(message, "<html") {
		re := regexp.MustCompile(`<title>([^<]+)</title>`)
		matches := re.FindStringSubmatch(message)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
		return ""
	}
	return message
}

// SanitizeAPIError detects if an error message contains HTML content (e.g., CloudFlare error pages)
// and returns a user-friendly message instead
func SanitizeAPIError(apiErr *APIError) string {
	message := apiErr.Message
	if message == "" {
		// Sometimes message is undefined
		// TODO: figure out why
		return ""
	}
	return sanitizeMessageHTML(message)
}

// FormatAPIError formats an API error into a user-friendly message
func FormatAPIError(err *APIError) string {
	// Extract connection error details from the cause chain
	connectionDetails := ExtractConnectionErrorDetails(err)

	if connectionDetails != nil {
		code := connectionDetails.Code
		isSSLError := connectionDetails.IsSSLError

		// Handle timeout errors
		if code == "ETIMEDOUT" {
			return "Request timed out. Check your internet connection and proxy settings"
		}

		// Handle SSL/TLS errors with specific messages
		if isSSLError {
			switch code {
			case "UNABLE_TO_VERIFY_LEAF_SIGNATURE", "UNABLE_TO_GET_ISSUER_CERT", "UNABLE_TO_GET_ISSUER_CERT_LOCALLY":
				return "Unable to connect to API: SSL certificate verification failed. Check your proxy or corporate SSL certificates"
			case "CERT_HAS_EXPIRED":
				return "Unable to connect to API: SSL certificate has expired"
			case "CERT_REVOKED":
				return "Unable to connect to API: SSL certificate has been revoked"
			case "DEPTH_ZERO_SELF_SIGNED_CERT", "SELF_SIGNED_CERT_IN_CHAIN":
				return "Unable to connect to API: Self-signed certificate detected. Check your proxy or corporate SSL certificates"
			case "ERR_TLS_CERT_ALTNAME_INVALID", "HOSTNAME_MISMATCH":
				return "Unable to connect to API: SSL certificate hostname mismatch"
			case "CERT_NOT_YET_VALID":
				return "Unable to connect to API: SSL certificate is not yet valid"
			default:
				return "Unable to connect to API: SSL error (" + code + ")"
			}
		}
	}

	if err.Message == "Connection error." {
		// If we have a code but it's not SSL, include it for debugging
		if connectionDetails != nil && connectionDetails.Code != "" {
			return "Unable to connect to API (" + connectionDetails.Code + ")"
		}
		return "Unable to connect to API. Check your internet connection"
	}

	// Guard: when deserialized from JSONL (e.g. --resume), the error object may
	// be a plain object without a message property. Return a safe fallback
	// instead of undefined, which would crash callers that access length.
	if err.Message == "" {
		nestedMsg := extractNestedErrorMessage(err)
		if nestedMsg != "" {
			return nestedMsg
		}
		status := "unknown"
		if err.Status != 0 {
			status = string(rune(err.Status))
		}
		return "API error (status " + status + ")"
	}

	sanitizedMessage := SanitizeAPIError(err)
	// Use sanitized message if it's different from the original (i.e., HTML was sanitized)
	if sanitizedMessage != err.Message && len(sanitizedMessage) > 0 {
		return sanitizedMessage
	}
	return err.Message
}

// extractNestedErrorMessage extracts a human-readable message from a deserialized API error
// that lacks a top-level message.
//
// Checks two nesting levels (deeper first for specificity):
// 1. error.error.error.message — standard Anthropic API shape
// 2. error.error.message — Bedrock shape
func extractNestedErrorMessage(err *APIError) string {
	if err.NestedError == nil {
		return ""
	}

	// Standard Anthropic API shape: { error: { error: { message } } }
	if err.NestedError.Error != nil && err.NestedError.Error.Message != "" {
		sanitized := sanitizeMessageHTML(err.NestedError.Error.Message)
		if len(sanitized) > 0 {
			return sanitized
		}
	}

	// Bedrock shape: { error: { message } }
	if err.NestedError.Message != "" {
		sanitized := sanitizeMessageHTML(err.NestedError.Message)
		if len(sanitized) > 0 {
			return sanitized
		}
	}

	return ""
}
