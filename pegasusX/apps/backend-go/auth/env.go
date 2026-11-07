package auth

import (
	"os"
	"strings"
)

// Deployment env classes. PEGASUSX_ENV=ssmr is a deprecated alias for sandbox.
const (
	EnvClassLocal      = "local"
	EnvClassSandbox    = "sandbox"
	EnvClassStaging    = "staging"
	EnvClassProduction = "production"
)

// EnvClassFrom maps a raw PEGASUSX_ENV value onto a canonical class.
//
//	sandbox | ssmr          → sandbox
//	staging                 → staging
//	production | prod       → production
//	empty | dev | local | * → local
func EnvClassFrom(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sandbox", "ssmr":
		return EnvClassSandbox
	case "staging":
		return EnvClassStaging
	case "production", "prod":
		return EnvClassProduction
	default:
		return EnvClassLocal
	}
}

// EnvClass is EnvClassFrom(PEGASUSX_ENV).
func EnvClass() string {
	return EnvClassFrom(os.Getenv("PEGASUSX_ENV"))
}

// IsSandbox is true for PEGASUSX_ENV=sandbox and the ssmr alias.
func IsSandbox() bool {
	return EnvClass() == EnvClassSandbox
}

// IsStaging is true for PEGASUSX_ENV=staging.
func IsStaging() bool {
	return EnvClass() == EnvClassStaging
}

// IsProduction is true for PEGASUSX_ENV=production|prod.
func IsProduction() bool {
	return EnvClass() == EnvClassProduction
}

// IsEnforcedEnv is sandbox or production (tenant + seed fail-closed defaults).
func IsEnforcedEnv() bool {
	return IsSandbox() || IsProduction()
}
