// Package foundryroutes mounts HTTP endpoints for the PegasusX Foundry Semantic-Kinetic-Dynamic
// Triad, Role 7 (Depot Cashier & AML Vault), Role 9 (Fleet, Yard & DVIR Red-Tag), and the
// 12 Cross-Role Pre-Execution Edge-Case Guards.
package foundryroutes

import (
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

// Deps bundles dependencies for the Foundry HTTP router.
type Deps struct {
	Engine *foundry.Engine
}

// RegisterRoutes mounts all /v1/foundry/* endpoints onto the chi router.
// Deconstructs routing into domain handlers (ontology, cashier, fleet, guards) while preserving
// exact backward-compatible URL patterns and responses.
func RegisterRoutes(r chi.Router, deps Deps) {
	eng := deps.Engine
	if eng == nil {
		eng = foundry.NewEngine()
	}

	r.Group(func(gr chi.Router) {
		gr.Use(requireFoundrySession)
		newOntologyHandler(eng).mountRoutes(gr)
		newCashierHandler(eng).mountRoutes(gr)
		newFleetHandler(eng).mountRoutes(gr)
		newGuardsHandler(eng).mountRoutes(gr)
	})
}
