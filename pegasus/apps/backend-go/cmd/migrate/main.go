// cmd/migrate is the out-of-band Spanner DDL migration runner.
//
// In production, the backend is deployed with MIGRATE_ON_BOOT=false so that
// pods do NOT race UpdateDatabaseDdl on cold start. This binary is invoked
// once per release (via Cloud Run Job, Cloud Build step, or operator shell)
// to apply pending DDL atomically before pods roll out.
//
// In dev / CI, MIGRATE_ON_BOOT defaults to "true" and main.go invokes the
// same migrations.Run() in-process — no need to run this binary manually.
//
// Usage:
//
//	cd pegasus/apps/backend-go
//	go run ./cmd/migrate              # uses cfg + Spanner from env (same as backend)
//	go run ./cmd/migrate -dry-run     # print intent, do not execute (TODO)
package main

import (
	"context"
	"flag"
	"log"

	"backend-go/bootstrap"
	"backend-go/migrations"

	"config"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "print which migrations would run, do not execute (not yet implemented)")
	flag.Parse()

	if *dryRun {
		log.Fatalln("[migrate] -dry-run not yet implemented; bail out and inspect migrations/migrations.go directly")
	}

	log.Println("[migrate] loading config")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[migrate] config load: %v", err)
	}

	ctx := context.Background()

	log.Println("[migrate] bootstrapping Spanner client + admin opts")
	app, err := bootstrap.NewApp(ctx, cfg)
	if err != nil {
		log.Fatalf("[migrate] bootstrap: %v", err)
	}
	defer app.Close()

	log.Println("[migrate] running migrations")
	migrations.Run(ctx, app.SpannerClientOpts, app.SpannerDBName, app.Spanner)
	log.Println("[migrate] done — migrations idempotent, safe to re-run")
}
