module github.com/pegasusx/pegasusx/apps/handoff-service

go 1.25.0

require (
	github.com/go-chi/chi/v5 v5.2.2
	github.com/pegasusx/pegasusx/packages/handoff v0.0.0
)

require github.com/google/uuid v1.6.0 // indirect

replace github.com/pegasusx/pegasusx/packages/handoff => ../../packages/handoff
