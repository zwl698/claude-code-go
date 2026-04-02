package constants

import "sync"

// SystemPromptSection represents a section of the system prompt
type SystemPromptSection struct {
	Name       string
	Compute    func() (string, error)
	CacheBreak bool
}

// SystemPromptSectionCache caches computed system prompt sections
type SystemPromptSectionCache struct {
	mu    sync.RWMutex
	cache map[string]string
}

var globalCache = &SystemPromptSectionCache{
	cache: make(map[string]string),
}

// SystemPromptSection creates a memoized system prompt section
func SystemPromptSectionNew(name string, compute func() (string, error)) SystemPromptSection {
	return SystemPromptSection{
		Name:       name,
		Compute:    compute,
		CacheBreak: false,
	}
}

// DangerousUncachedSystemPromptSection creates a volatile system prompt section
// that recomputes every turn. This WILL break the prompt cache when the value changes.
func DangerousUncachedSystemPromptSection(name string, compute func() (string, error), _reason string) SystemPromptSection {
	return SystemPromptSection{
		Name:       name,
		Compute:    compute,
		CacheBreak: true,
	}
}

// ResolveSystemPromptSections resolves all system prompt sections, returning prompt strings
func ResolveSystemPromptSections(sections []SystemPromptSection) ([]string, error) {
	results := make([]string, len(sections))

	for i, section := range sections {
		if !section.CacheBreak {
			// Check cache first
			if val, ok := globalCache.Get(section.Name); ok {
				results[i] = val
				continue
			}
		}

		// Compute the section
		val, err := section.Compute()
		if err != nil {
			return nil, err
		}

		// Cache if not cache-breaking
		if !section.CacheBreak {
			globalCache.Set(section.Name, val)
		}

		results[i] = val
	}

	return results, nil
}

// ClearSystemPromptSections clears all system prompt section state
func ClearSystemPromptSections() {
	globalCache.Clear()
}

// Get retrieves a cached value
func (c *SystemPromptSectionCache) Get(name string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.cache[name]
	return val, ok
}

// Set stores a value in the cache
func (c *SystemPromptSectionCache) Set(name, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[name] = value
}

// Clear removes all entries from the cache
func (c *SystemPromptSectionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]string)
}
