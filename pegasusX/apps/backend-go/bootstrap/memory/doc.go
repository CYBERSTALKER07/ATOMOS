// Package memory provides in-memory repository scaffolds for local development
// and explicit opt-in fallback when Spanner adapters are unavailable.
//
// Memory stores are NEVER used when:
//   - PEGASUSX_ENV=production, or
//   - REQUIRE_INFRA_ADAPTERS=true (default), or
//   - ALLOW_MEMORY_FALLBACK is unset/false.
//
// Local/SSMR only: REQUIRE_INFRA_ADAPTERS=false ALLOW_MEMORY_FALLBACK=true.
// Bootstrap fails closed otherwise — no silent production memory path.
package memory

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// OutboxAppender appends outbox events in scaffold / fallback mode.
type OutboxAppender interface {
	Append(ctx context.Context, events []outbox.Event) error
}
