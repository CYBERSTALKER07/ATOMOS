// Package memory provides in-memory repository scaffolds for local development
// and fallback when Spanner adapters are unavailable (REQUIRE_INFRA_ADAPTERS=false).
package memory

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// OutboxAppender appends outbox events in scaffold / fallback mode.
type OutboxAppender interface {
	Append(ctx context.Context, events []outbox.Event) error
}
