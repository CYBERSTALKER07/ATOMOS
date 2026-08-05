package bootstrap

import (
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

// OutboundCircuits holds outbound circuit breakers for external dependencies.
type OutboundCircuits struct {
	Payment      *circuit.Breaker
	Notification *circuit.Breaker
	Telegram     *circuit.Breaker
	OSRM         *circuit.Breaker
	GoogleRoutes *circuit.Breaker
}

// NewOutboundCircuits constructs default outbound breakers.
func NewOutboundCircuits() *OutboundCircuits {
	cfg := circuit.Config{}
	return &OutboundCircuits{
		Payment:      circuit.New("payment", cfg),
		Notification: circuit.New("notification", cfg),
		Telegram:     circuit.New("telegram", cfg),
		OSRM:         circuit.New("osrm", cfg),
		GoogleRoutes: circuit.New("google_routes", cfg),
	}
}
