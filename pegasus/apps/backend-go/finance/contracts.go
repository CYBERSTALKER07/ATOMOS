package finance

const (
	// PlatformAccountID is the canonical platform account identifier for new ledger writes.
	PlatformAccountID = "ACC-PEGASUS"

	// LegacyPlatformAccountID is retained for historical rows and compatibility reads.
	LegacyPlatformAccountID = "ACC-PEGASUS"

	// PlatformCreditEntryType is the canonical platform-fee entry type for new writes.
	PlatformCreditEntryType = "CREDIT_PLATFORM"

	// LegacyPlatformCreditEntryType is retained for compatibility with existing rows.
	LegacyPlatformCreditEntryType = "CREDIT_PEGASUS"
)

// PlatformAccountIDsForQuery returns canonical + legacy platform account IDs.
func PlatformAccountIDsForQuery() []string {
	return []string{PlatformAccountID, LegacyPlatformAccountID}
}

// PlatformCreditEntryTypesForQuery returns canonical + legacy entry types.
// Historical values are included so analytics remain correct during migration.
func PlatformCreditEntryTypesForQuery() []string {
	return []string{
		PlatformCreditEntryType,
		LegacyPlatformCreditEntryType,
		"CREDIT",
		"COMMISSION",
	}
}

// IsPlatformAccount checks whether a ledger account belongs to the platform.
func IsPlatformAccount(accountID string) bool {
	return accountID == PlatformAccountID || accountID == LegacyPlatformAccountID
}
