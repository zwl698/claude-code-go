package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"claude-code-go/internal/constants"
	"claude-code-go/internal/utils"
)

// BootstrapResponse represents the response from the bootstrap API
type BootstrapResponse struct {
	ClientData             interface{}             `json:"client_data,omitempty"`
	AdditionalModelOptions []AdditionalModelOption `json:"additional_model_options,omitempty"`
}

// AdditionalModelOption represents an additional model option from bootstrap
type AdditionalModelOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// rawAdditionalModelOption represents the raw API response for model options
type rawAdditionalModelOption struct {
	Model       string `json:"model"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GlobalConfig represents the global configuration structure
// This is a simplified version - the actual implementation would have more fields
type GlobalConfig struct {
	ClientDataCache             interface{}             `json:"client_data_cache,omitempty"`
	AdditionalModelOptionsCache []AdditionalModelOption `json:"additional_model_options_cache,omitempty"`
}

// ConfigManager handles configuration management
// This is a placeholder - the actual implementation would be more complex
var configManager = &GlobalConfig{}

// fetchBootstrapAPI fetches bootstrap data from the API
func fetchBootstrapAPI(apiKey, oauthToken string, isSubscriber bool) (*BootstrapResponse, error) {
	// Check if only essential traffic is allowed
	if utils.IsEssentialTrafficOnly() {
		return nil, nil // Skip bootstrap in minimal/strict privacy mode
	}

	// Only fetch from first-party API
	if utils.GetAPIProvider() != utils.APIProviderFirstParty {
		return nil, nil
	}

	// OAuth preferred (requires user:profile scope — service-key OAuth tokens
	// lack it and would 403). Fall back to API key auth for console users.
	hasUsableOAuth := oauthToken != ""

	if !hasUsableOAuth && apiKey == "" {
		return nil, nil
	}

	oauthConfig := constants.GetOAuthConfig()
	endpoint := oauthConfig.BaseAPIURL + "/api/claude_cli/bootstrap"

	// Build auth headers
	var authHeaders map[string]string
	if hasUsableOAuth {
		authHeaders = map[string]string{
			"Authorization":  "Bearer " + oauthToken,
			"anthropic-beta": constants.OAuthBetaHeader,
		}
	} else if apiKey != "" {
		authHeaders = map[string]string{
			"x-api-key": apiKey,
		}
	} else {
		return nil, nil
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", utils.GetClaudeCodeUserAgent())
	for key, value := range authHeaders {
		req.Header.Set(key, value)
	}

	// Create client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse raw response
	var rawResponse struct {
		ClientData             interface{}                `json:"client_data,omitempty"`
		AdditionalModelOptions []rawAdditionalModelOption `json:"additional_model_options,omitempty"`
	}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Transform additional model options
	response := &BootstrapResponse{
		ClientData: rawResponse.ClientData,
	}

	for _, opt := range rawResponse.AdditionalModelOptions {
		response.AdditionalModelOptions = append(response.AdditionalModelOptions, AdditionalModelOption{
			Value:       opt.Model,
			Label:       opt.Name,
			Description: opt.Description,
		})
	}

	return response, nil
}

// FetchBootstrapData fetches bootstrap data from the API and persists to disk cache.
// This is a simplified version - the full implementation would:
// - Use withOAuth401Retry for token refresh
// - Save to global config file
// - Check for config changes before writing
func FetchBootstrapData(apiKey, oauthToken string, isSubscriber bool) error {
	response, err := fetchBootstrapAPI(apiKey, oauthToken, isSubscriber)
	if err != nil {
		return err
	}

	if response == nil {
		return nil
	}

	clientData := response.ClientData
	additionalModelOptions := response.AdditionalModelOptions
	if additionalModelOptions == nil {
		additionalModelOptions = []AdditionalModelOption{}
	}

	// Only persist if data actually changed — avoids a config write on every startup.
	if reflect.DeepEqual(configManager.ClientDataCache, clientData) &&
		reflect.DeepEqual(configManager.AdditionalModelOptionsCache, additionalModelOptions) {
		return nil
	}

	// Update config
	configManager.ClientDataCache = clientData
	configManager.AdditionalModelOptionsCache = additionalModelOptions

	// Persist to disk
	if err := saveGlobalConfig(configManager); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to save config: %v\n", err)
	}

	return nil
}

// GetClientDataCache returns the cached client data
func GetClientDataCache() interface{} {
	return configManager.ClientDataCache
}

// GetAdditionalModelOptionsCache returns the cached additional model options
func GetAdditionalModelOptionsCache() []AdditionalModelOption {
	return configManager.AdditionalModelOptionsCache
}

// ========================================
// Config Persistence
// ========================================

// saveGlobalConfig saves the global config to disk
func saveGlobalConfig(config *GlobalConfig) error {
	configPath := getGlobalConfigPath()

	// Ensure directory exists
	if err := utils.EnsureDir(configPath[:len(configPath)-len("/config.json")]); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := utils.WriteFileSync(configPath, string(data), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// loadGlobalConfig loads the global config from disk
func loadGlobalConfig() (*GlobalConfig, error) {
	configPath := getGlobalConfigPath()

	data := utils.ReadFileSafe(configPath)
	if data == "" {
		return &GlobalConfig{}, nil
	}

	var config GlobalConfig
	if err := json.Unmarshal([]byte(data), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// getGlobalConfigPath returns the path to the global config file
func getGlobalConfigPath() string {
	configHome := utils.GetClaudeConfigHome()
	return configHome + "/config.json"
}

// init initializes the config manager
func init() {
	// Load existing config on startup
	if config, err := loadGlobalConfig(); err == nil && config != nil {
		configManager = config
	}
}
