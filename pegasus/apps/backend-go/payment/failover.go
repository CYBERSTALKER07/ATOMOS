package payment

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend-go/cache"
)

const gatewayHealthKeyPrefix = "gw:health:"

// RecordGatewayOutcome tracks rolling success/failure for payment gateway failover.
func RecordGatewayOutcome(ctx context.Context, gateway string, success bool) {
	if cache.GetClient() == nil {
		return
	}
	key := gatewayHealthKeyPrefix + normalizeGateway(gateway)
	field := "failure"
	if success {
		field = "success"
	}
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	_ = cache.GetClient().HIncrBy(ctx, key, field, 1).Err()
	_ = cache.GetClient().Expire(ctx, key, 24*time.Hour).Err()
}

// IsGatewayHealthy returns false when failure rate exceeds 50% with at least 5 attempts.
func IsGatewayHealthy(ctx context.Context, gateway string) bool {
	if cache.GetClient() == nil {
		return true
	}
	key := gatewayHealthKeyPrefix + normalizeGateway(gateway)
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	vals, err := cache.GetClient().HGetAll(ctx, key).Result()
	if err != nil || len(vals) == 0 {
		return true
	}
	var success, failure int64
	fmt.Sscan(vals["success"], &success)
	fmt.Sscan(vals["failure"], &failure)
	total := success + failure
	if total < 5 {
		return true
	}
	return failure*2 < total
}

// SelectHealthyGateway picks the first healthy gateway from policy order.
func SelectHealthyGateway(ctx context.Context, gateways []string) string {
	for _, g := range gateways {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if IsGatewayHealthy(ctx, g) {
			return g
		}
	}
	if len(gateways) > 0 {
		return strings.TrimSpace(gateways[0])
	}
	return ""
}

// IsFailoverRetryable classifies provider errors eligible for gateway failover.
func IsFailoverRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "503") ||
		strings.Contains(msg, "502") ||
		strings.Contains(msg, "504") ||
		strings.Contains(msg, "500") ||
		isServerError(err)
}

// ResolveWithFailover returns the execution client for the first healthy gateway.
func (r *ProviderExecutionRouter) ResolveWithFailover(ctx context.Context, gateways []string, credsFactory func(gateway string) (ProviderExecutionCredentials, error)) (ProviderExecutionClient, string, error) {
	if r == nil {
		return nil, "", fmt.Errorf("provider execution router not configured")
	}
	if len(gateways) == 0 {
		return nil, "", fmt.Errorf("no gateways in policy")
	}
	var lastErr error
	for _, gateway := range gateways {
		gateway = normalizeGateway(gateway)
		if gateway == "" || !IsGatewayHealthy(ctx, gateway) {
			continue
		}
		creds, err := credsFactory(gateway)
		if err != nil {
			lastErr = err
			continue
		}
		client, err := r.Resolve(gateway, creds)
		if err != nil {
			lastErr = err
			continue
		}
		return client, gateway, nil
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("no healthy gateway available")
}

// statusCoder is implemented by HTTP-aware provider errors.
type statusCoder interface {
	StatusCode() int
}

func isServerError(err error) bool {
	if sc, ok := err.(statusCoder); ok {
		return sc.StatusCode() >= http.StatusInternalServerError
	}
	return false
}
