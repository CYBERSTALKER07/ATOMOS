package promotion

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A stub spanner.Client mock would be needed here, or we use the MemoryRepository pattern if applicable.
// Since these functions take *spanner.Client directly, testing requires a spanner emulator or we skip full integration in unit tests.
// Let's create a placeholder test that validates the logic we can.

func TestCampaignCreation(t *testing.T) {
	_, err := CreateCampaign(context.Background(), nil, "sup-1", "Test Promo", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "budget must be greater than 0")
}
