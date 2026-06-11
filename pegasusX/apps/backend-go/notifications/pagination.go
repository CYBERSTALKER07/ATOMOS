package notifications

import (
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/httppagination"
)

const (
	// DefaultInboxLimit is the inbox page size when the client omits ?limit=.
	DefaultInboxLimit = 50
	// MaxInboxLimit caps a single inbox fetch to protect Spanner and mobile payloads.
	MaxInboxLimit = 500
)

// ParseInboxPagination reads ?limit=&offset= for GET /v1/user/notifications.
func ParseInboxPagination(r *http.Request) (limit, offset int) {
	return httppagination.ParseLimitOffset(r, DefaultInboxLimit, MaxInboxLimit)
}
