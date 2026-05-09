import re

with open('pegasus/apps/backend-go/workers/utility_meter.go', 'r') as f:
    content = f.read()

content = content.replace('"ENTRY_TYPE_INFRA_OVERAGE", "ledger", OverageEvent{', '"ledger", OverageEvent{')

content = re.sub(
        r'},\s*telemetry\.TraceIDFromContext\(ctx\)\)\n\s*}\)\n\n\s*if err != nil {',
        '}))\n\t})\n\n\t\t\t\tif err != nil {',
        content
)

# Actually let's just do a clean replace of the body of chargeOverage.
replacement = """func (u *UtilityMeter) chargeOverage(ctx context.Context, supplierID string, minorUnits int64) {
        _, err := u.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
                entryID := spanner.CommitTimestamp.String()
                
                var muts []*spanner.Mutation
                
                muts = append(muts, spanner.Insert("LedgerEntries",
                        []string{"EntryId", "AccountId", "Amount", "Currency", "EntryType", "CreatedAt", "Status"},
                        []interface{}{entryID + "-debit", "supplier:" + supplierID + ":wallet", -minorUnits, "USD", "ENTRY_TYPE_INFRA_OVERAGE", spanner.CommitTimestamp, "SETTLED"},
                ))
                
                muts = append(muts, spanner.Insert("LedgerEntries",
                        []string{"EntryId", "AccountId", "Amount", "Currency", "EntryType", "CreatedAt", "Status"},
                        []interface{}{entryID + "-credit", "platform:fee", minorUnits, "USD", "ENTRY_TYPE_INFRA_OVERAGE", spanner.CommitTimestamp, "SETTLED"},
                ))
                
                if err := txn.BufferWrite(muts); err != nil {
                        return err
                }
                
                type OverageEvent struct {
                        SupplierID string `json:"supplier_id"`
                        Amount     int64  `json:"amount"`
                        Currency   string `json:"currency"`
                }
                return outbox.EmitJSON(txn, "Ledger", supplierID, "ENTRY_TYPE_INFRA_OVERAGE", "ledger", OverageEvent{
                        SupplierID: supplierID,
                        Amount:     minorUnits,
                        Currency:   "USD",
                }, telemetry.TraceIDFromContext(ctx))
        })

        if err != nil {
                slog.ErrorContext(ctx, "utility_meter.charge_failed", "supplier_id", supplierID, "err", err)
        } else {
                slog.InfoContext(ctx, "utility_meter.charge_success", "supplier_id", supplierID, "amount_cents", minorUnits)
        }
}
"""

content = re.sub(r'func \(u \*UtilityMeter\) chargeOverage[\s\S]*?^}$', replacement, content, flags=re.MULTILINE)

with open('pegasus/apps/backend-go/workers/utility_meter.go', 'w') as f:
    f.write(content)
