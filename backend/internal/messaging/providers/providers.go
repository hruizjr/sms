// Package providers builds the configured provider adapter. It is the only place that knows the
// concrete adapters; the messaging engine depends on the provider interface alone, so adding a
// provider changes this package and its configuration, never the engine or the API.
package providers

import (
	"fmt"

	"github.com/hruizjr/sms/backend/internal/config"
	"github.com/hruizjr/sms/backend/internal/messaging/provider"
	"github.com/hruizjr/sms/backend/internal/messaging/provider/simulator"
)

// New returns the adapter selected by cfg.Name.
func New(cfg config.ProviderConfig) (provider.Provider, error) {
	switch cfg.Name {
	case config.ProviderSimulator:
		return simulator.New(simulator.Config{
			ReceiptDelay: cfg.Simulator.ReceiptDelay,
			MaxSegments:  cfg.Simulator.MaxSegments,
		}), nil
	case "":
		return nil, fmt.Errorf("no SMS provider is configured (%s)", config.EnvSMSProvider)
	}
	return nil, fmt.Errorf("unknown SMS provider %q", cfg.Name)
}
