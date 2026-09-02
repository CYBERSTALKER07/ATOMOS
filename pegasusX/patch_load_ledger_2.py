import re

with open("apps/backend-go/stocklots/load_ledger.go", "r") as f:
    content = f.read()

append_str = """
func memoryBlocked() error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "production") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "sandbox") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_INFRA_ADAPTERS")), "true") {
		return fmt.Errorf("in-memory load ledger blocked in production/infra mode")
	}
	return nil
}
"""

content = content.replace('// memoryLoadLedger is used when Spanner is unavailable (tests / demo overlay).', append_str + '\n// memoryLoadLedger is used when Spanner is unavailable (tests / demo overlay).')

with open("apps/backend-go/stocklots/load_ledger.go", "w") as f:
    f.write(content)

