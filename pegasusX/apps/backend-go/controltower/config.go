package controltower

import (
	"os"
	"strings"
)

// Config holds feature flags for O9-2 playbooks.
type Config struct {
	Enabled     bool
	AutoExecute bool
}

// ConfigFromEnv reads CONTROL_TOWER_PLAYBOOKS_* flags (default off).
func ConfigFromEnv() Config {
	return Config{
		Enabled:     envBool("CONTROL_TOWER_PLAYBOOKS_ENABLED", false),
		AutoExecute: envBool("CONTROL_TOWER_PLAYBOOKS_AUTO_EXECUTE", false),
	}
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return strings.EqualFold(v, "true") || v == "1"
}
