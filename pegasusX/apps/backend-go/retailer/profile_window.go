package retailer

import (
	"errors"
	"fmt"

	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
)

func validateReceivingWindowField(raw string) (string, error) {
	canon, err := proximity.ValidateReceivingWindow(raw)
	if err != nil {
		if errors.Is(err, proximity.ErrInvalidReceivingWindow) {
			return "", fmt.Errorf("invalid receiving window: expected HH:MM 24-hour format")
		}
		return "", err
	}
	return canon, nil
}
