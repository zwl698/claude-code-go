// Package api provides API client functionality for the claude-code CLI.
// This file contains AWS SigV4 signing utilities for Bedrock requests.
package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// AWS signing constants
const (
	awsAlgorithm     = "AWS4-HMAC-SHA256"
	awsService       = "bedrock"
	awsRequest       = "aws4_request"
	aws4Header       = "X-Amz-Date"
	awsSecurity      = "X-Amz-Security-Token"
	awsAuthorization = "Authorization"
)

// AWSSigner provides AWS SigV4 signing functionality.
type AWSSigner struct {
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
	region          string
}

// NewAWSSigner creates a new AWS signer.
func NewAWSSigner(creds *AWSCredentials, region string) *AWSSigner {
	if creds == nil {
		return &AWSSigner{region: region}
	}
	return &AWSSigner{
		accessKeyID:     creds.AccessKeyID,
		secretAccessKey: creds.SecretAccessKey,
		sessionToken:    creds.SessionToken,
		region:          region,
	}
}

// SignRequest signs an HTTP request with AWS SigV4.
func (s *AWSSigner) SignRequest(req *http.Request) error {
	// If no credentials, skip signing
	if s.accessKeyID == "" || s.secretAccessKey == "" {
		return nil
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	// Set required headers
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set(aws4Header, amzDate)

	if s.sessionToken != "" {
		req.Header.Set(awsSecurity, s.sessionToken)
	}

	// Create canonical request
	canonicalRequest := s.createCanonicalRequest(req)

	// Create string to sign
	stringToSign := s.createStringToSign(amzDate, dateStamp, canonicalRequest)

	// Calculate signature
	signingKey := s.getSignatureKey(dateStamp)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	// Create authorization header
	credentialScope := fmt.Sprintf("%s/%s/%s/%s",
		dateStamp, s.region, awsService, awsRequest)
	signedHeaders := s.getSignedHeaders(req)

	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		awsAlgorithm, s.accessKeyID, credentialScope, signedHeaders, signature)

	req.Header.Set(awsAuthorization, authorization)

	return nil
}

// createCanonicalRequest creates the canonical request string.
func (s *AWSSigner) createCanonicalRequest(req *http.Request) string {
	// HTTP method
	method := req.Method

	// Canonical URI
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	// Canonical query string
	canonicalQueryString := s.getCanonicalQueryString(req.URL.Query())

	// Canonical headers
	canonicalHeaders, _ := s.getCanonicalHeaders(req)

	// Signed headers
	signedHeaders := s.getSignedHeaders(req)

	// Payload hash (for requests with body, we'd hash the body)
	payloadHash := "UNSIGNED-PAYLOAD"
	if req.Body == nil {
		// For GET requests or empty body
		hash := sha256.Sum256([]byte(""))
		payloadHash = hex.EncodeToString(hash[:])
	}

	return strings.Join([]string{
		method,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		"",
		signedHeaders,
		payloadHash,
	}, "\n")
}

// getCanonicalQueryString creates the canonical query string.
func (s *AWSSigner) getCanonicalQueryString(query url.Values) string {
	if len(query) == 0 {
		return ""
	}

	// Sort keys
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		for _, v := range query[k] {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	return strings.Join(parts, "&")
}

// getCanonicalHeaders creates the canonical headers string.
func (s *AWSSigner) getCanonicalHeaders(req *http.Request) (string, []string) {
	// Collect headers
	headers := make(map[string][]string)
	var headerNames []string

	for k, v := range req.Header {
		lowerKey := strings.ToLower(k)
		if lowerKey == "authorization" || lowerKey == "user-agent" {
			continue
		}
		headers[lowerKey] = v
		headerNames = append(headerNames, lowerKey)
	}

	// Sort header names
	sort.Strings(headerNames)

	// Build canonical headers string
	var parts []string
	for _, name := range headerNames {
		for _, value := range headers[name] {
			parts = append(parts, name+":"+strings.TrimSpace(value))
		}
	}

	return strings.Join(parts, "\n") + "\n", headerNames
}

// getSignedHeaders returns the signed headers string.
func (s *AWSSigner) getSignedHeaders(req *http.Request) string {
	var headerNames []string

	for k := range req.Header {
		lowerKey := strings.ToLower(k)
		if lowerKey == "authorization" || lowerKey == "user-agent" {
			continue
		}
		headerNames = append(headerNames, lowerKey)
	}

	sort.Strings(headerNames)
	return strings.Join(headerNames, ";")
}

// createStringToSign creates the string to sign.
func (s *AWSSigner) createStringToSign(amzDate, dateStamp, canonicalRequest string) string {
	credentialScope := fmt.Sprintf("%s/%s/%s/%s",
		dateStamp, s.region, awsService, awsRequest)

	hash := sha256.Sum256([]byte(canonicalRequest))

	return strings.Join([]string{
		awsAlgorithm,
		amzDate,
		credentialScope,
		hex.EncodeToString(hash[:]),
	}, "\n")
}

// getSignatureKey derives the signing key.
func (s *AWSSigner) getSignatureKey(dateStamp string) []byte {
	kDate := hmacSHA256Raw([]byte("AWS4"+s.secretAccessKey), dateStamp)
	kRegion := hmacSHA256Raw(kDate, s.region)
	kService := hmacSHA256Raw(kRegion, awsService)
	kSigning := hmacSHA256Raw(kService, awsRequest)
	return kSigning
}

// hmacSHA256 computes HMAC-SHA256 and returns hex string.
func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

// hmacSHA256Raw computes HMAC-SHA256 and returns raw bytes.
func hmacSHA256Raw(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
