package bootstrap

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/infraroutes"
)

func buildInfraHealthChecks(redisEnabled bool, cacheBackend cache.Backend, spannerClient *spanner.Client) infraroutes.Deps {
	checks := make(map[string]infraroutes.HealthChecker)
	if redisEnabled {
		if pinger, ok := cacheBackend.(interface {
			Ping(ctx context.Context) error
		}); ok {
			checks["redis"] = infraroutes.HealthCheckFunc(pinger.Ping)
		}
	}
	if spannerClient != nil {
		client := spannerClient
		checks["spanner"] = infraroutes.HealthCheckFunc(func(ctx context.Context) error {
			iter := client.Single().Query(ctx, spanner.Statement{SQL: "SELECT 1"})
			defer iter.Stop()
			_, err := iter.Next()
			return err
		})
	}
	return infraroutes.Deps{Checks: checks}
}
