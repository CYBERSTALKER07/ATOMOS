import re

with open("apps/backend-go/payload/ship_units.go", "r") as f:
    content = f.read()

pattern = re.compile(r'err = s\.repo\.RunTx\(ctx, func\(ctx context\.Context, tx PayloadTx\) error \{\n\s*// Use spanner transaction if possible.*?return nil\n\t\}, func\(txn outbox\.TxnBuffer\) error \{', re.DOTALL)

replacement = r"""err = s.repo.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
		if err := tx.SaveShipUnits(ctx, newUnits); err != nil {
			return err
		}
		// Fallback to in-memory
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.shipUnits == nil {
			s.shipUnits = map[string][]ShipUnit{}
		}
		s.shipUnits[manifestID] = append(s.shipUnits[manifestID], newUnits...)
		return nil
	}, func(txn outbox.TxnBuffer) error {"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/payload/ship_units.go", "w") as f:
    f.write(content)
