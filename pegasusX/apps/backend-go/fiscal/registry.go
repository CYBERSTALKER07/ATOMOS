package fiscal

import (
	"errors"
	"sync"
)

var (
	ErrStrategyNotFound          = errors.New("fiscal strategy not found for country code")
	ErrStrategyAlreadyRegistered = errors.New("fiscal strategy already registered for country code")
)

// Registry manages fiscal strategies by country code.
type Registry struct {
	mu         sync.RWMutex
	strategies map[string]FiscalStrategy
}

// NewRegistry creates a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		strategies: make(map[string]FiscalStrategy),
	}
}

// Register adds a new FiscalStrategy for a specific country code.
func (r *Registry) Register(countryCode string, strategy FiscalStrategy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.strategies[countryCode]; exists {
		return ErrStrategyAlreadyRegistered
	}
	r.strategies[countryCode] = strategy
	return nil
}

// Get retrieves the FiscalStrategy for a specific country code.
func (r *Registry) Get(countryCode string) (FiscalStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if strategy, exists := r.strategies[countryCode]; exists {
		return strategy, nil
	}
	return nil, ErrStrategyNotFound
}

// globalRegistry provides a convenient package-level registry.
var globalRegistry = NewRegistry()

// Register adds a strategy to the global registry.
func Register(countryCode string, strategy FiscalStrategy) error {
	return globalRegistry.Register(countryCode, strategy)
}

// Get retrieves a strategy from the global registry.
func Get(countryCode string) (FiscalStrategy, error) {
	return globalRegistry.Get(countryCode)
}
