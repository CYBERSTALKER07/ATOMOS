import re

with open("apps/backend-go/stocklots/load_ledger.go", "r") as f:
    content = f.read()

pattern1 = re.compile(r'func SeedLoadLedgerMemory\(manifestID string, lines \[\]SeedLoadLine\) \{')
replacement1 = r"""func memoryBlocked() error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "production") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "sandbox") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_INFRA_ADAPTERS")), "true") {
		return fmt.Errorf("in-memory load ledger blocked in production/infra mode")
	}
	return nil
}

// SeedLoadLedgerMemory seeds memory ledger (tests).
func SeedLoadLedgerMemory(manifestID string, lines []SeedLoadLine) error {
	if err := memoryBlocked(); err != nil {
		return err
	}"""
content = pattern1.sub(replacement1, content)

pattern2 = re.compile(r'func ScanLoadLineMemory\(manifestID, orderID, lineOrSku string, delta int64\) \(\*LoadLine, error\) \{')
replacement2 = r"""func ScanLoadLineMemory(manifestID, orderID, lineOrSku string, delta int64) (*LoadLine, error) {
	if err := memoryBlocked(); err != nil {
		return nil, err
	}"""
content = pattern2.sub(replacement2, content)

pattern3 = re.compile(r'func ApproveLoadVarianceMemory\(manifestID, orderID, lineID string\) \(\*LoadLine, error\) \{')
replacement3 = r"""func ApproveLoadVarianceMemory(manifestID, orderID, lineID string) (*LoadLine, error) {
	if err := memoryBlocked(); err != nil {
		return nil, err
	}"""
content = pattern3.sub(replacement3, content)

pattern4 = re.compile(r'func ListLoadLedgerMemory\(manifestID string\) \[\]LoadLine \{')
replacement4 = r"""func ListLoadLedgerMemory(manifestID string) []LoadLine {
	if err := memoryBlocked(); err != nil {
		return nil
	}"""
content = pattern4.sub(replacement4, content)

with open("apps/backend-go/stocklots/load_ledger.go", "w") as f:
    f.write(content)

