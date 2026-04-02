package api

// NonNullableUsage represents API usage information that is guaranteed to be non-null
type NonNullableUsage struct {
	InputTokens              int             `json:"input_tokens"`
	CacheCreationInputTokens int             `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int             `json:"cache_read_input_tokens"`
	OutputTokens             int             `json:"output_tokens"`
	ServerToolUse            ServerToolUsage `json:"server_tool_use"`
	ServiceTier              string          `json:"service_tier"`
	CacheCreation            CacheCreation   `json:"cache_creation"`
	InferenceGeo             string          `json:"inference_geo"`
	Iterations               []Iteration     `json:"iterations"`
	Speed                    string          `json:"speed"`
}

// ServerToolUsage represents server-side tool usage statistics
type ServerToolUsage struct {
	WebSearchRequests int `json:"web_search_requests"`
	WebFetchRequests  int `json:"web_fetch_requests"`
}

// CacheCreation represents cache creation token details
type CacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
}

// Iteration represents a single iteration in the API response
type Iteration struct {
	// Add fields as needed based on actual API response
}

// EMPTY_USAGE is a zero-initialized usage object.
// Extracted from logging logic so that other packages can import it
// without transitively pulling in many dependencies.
var EMPTY_USAGE = NonNullableUsage{
	InputTokens:              0,
	CacheCreationInputTokens: 0,
	CacheReadInputTokens:     0,
	OutputTokens:             0,
	ServerToolUse: ServerToolUsage{
		WebSearchRequests: 0,
		WebFetchRequests:  0,
	},
	ServiceTier: "standard",
	CacheCreation: CacheCreation{
		Ephemeral1hInputTokens: 0,
		Ephemeral5mInputTokens: 0,
	},
	InferenceGeo: "",
	Iterations:   []Iteration{},
	Speed:        "standard",
}
