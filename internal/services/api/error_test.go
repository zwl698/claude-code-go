package api

import (
	"testing"
)

func TestAPIErrorWithNestedError(t *testing.T) {
	err := &APIError{
		Status:  401,
		Message: "Unauthorized",
		NestedError: &NestedError{
			Message: "OAuth token has been revoked",
		},
	}

	expected := "API Error: 401 Unauthorized"
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}

	formatted := FormatAPIError(err)
	if formatted == "" {
		t.Error("FormatAPIError returned empty string")
	}

	classification := ClassifyAPIError(err)
	if classification == "" {
		t.Error("ClassifyAPIError returned empty string")
	}
}

func TestAPIErrorWithoutMessage(t *testing.T) {
	err := &APIError{
		Status: 500,
		NestedError: &NestedError{
			Message: "Internal server error",
		},
	}

	errorStr := err.Error()
	if errorStr == "" {
		t.Error("Error() returned empty string")
	}
}

func TestEmptyUsage(t *testing.T) {
	usage := EMPTY_USAGE
	if usage.InputTokens != 0 {
		t.Errorf("Expected InputTokens to be 0, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 0 {
		t.Errorf("Expected OutputTokens to be 0, got %d", usage.OutputTokens)
	}
}

func TestExtractNestedErrorMessage(t *testing.T) {
	// Test standard Anthropic API shape
	err := &APIError{
		Status: 500,
		NestedError: &NestedError{
			Error: &NestedError{
				Message: "Deep nested error",
			},
		},
	}

	msg := extractNestedErrorMessage(err)
	if msg != "Deep nested error" {
		t.Errorf("Expected 'Deep nested error', got '%s'", msg)
	}

	// Test Bedrock shape
	err2 := &APIError{
		Status: 500,
		NestedError: &NestedError{
			Message: "Bedrock error",
		},
	}

	msg2 := extractNestedErrorMessage(err2)
	if msg2 != "Bedrock error" {
		t.Errorf("Expected 'Bedrock error', got '%s'", msg2)
	}
}
