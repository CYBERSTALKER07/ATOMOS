import re

with open("apps/backend-go/payload/ship_units.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(s \*Service\) EnsureShipUnitsForManifest\(ctx context\.Context, manifestID string\) \(int, error\) \{.*?return n, nil\n\}', re.DOTALL)

replacement = r"""func (s *Service) EnsureShipUnitsForManifest(ctx context.Context, manifestID string) (int, error) {
	if s == nil || !gs1.LabelsEnabled() {
		return 0, nil
	}
	manifestID = strings.TrimSpace(manifestID)
	if manifestID == "" {
		return 0, fmt.Errorf("manifest_id_required")
	}
	prefix, err := s.loadGs1CompanyPrefix(ctx)
	if err != nil || strings.TrimSpace(prefix) == "" {
		if s.log != nil {
			s.log.Info("gs1 sscc skipped: no company prefix", "manifest_id", manifestID, "err", err)
		}
		return 0, nil
	}
	orderIDs := s.manifestOrderIDs(ctx, manifestID)
	if len(orderIDs) == 0 {
		return 0, nil
	}
	existing, _ := s.ListShipUnits(ctx, manifestID)
	have := map[string]bool{}
	for _, u := range existing {
		have[u.OrderID] = true
	}
	
	newUnits := []ShipUnit{}
	seq := int64(len(existing))
	for _, oid := range orderIDs {
		if have[oid] {
			continue
		}
		serial := ssccSerial(manifestID, oid, seq)
		sscc, err := gs1.GenerateSSCC(prefix, serial)
		if err != nil {
			return 0, err
		}
		unit := ShipUnit{
			ManifestID: manifestID,
			ShipUnitID: uuid.NewString(),
			SSCC:       sscc,
			OrderID:    oid,
			Sequence:   seq,
			CreatedAt:  s.now().UTC(),
		}
		newUnits = append(newUnits, unit)
		seq++
	}
	
	if len(newUnits) == 0 {
		return 0, nil
	}
	
	err = s.repo.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
		// Use spanner transaction if possible
		if sp, ok := tx.(interface{ Txn() *spanner.ReadWriteTransaction }); ok {
			txn := sp.Txn()
			if txn != nil {
				var muts []*spanner.Mutation
				for _, u := range newUnits {
					muts = append(muts, spanner.InsertMap("ManifestShipUnits", map[string]any{
						"ManifestId": u.ManifestID,
						"ShipUnitId": u.ShipUnitID,
						"Sscc":       u.SSCC,
						"OrderId":    u.OrderID,
						"Sequence":   u.Sequence,
						"Gtin":       nullableStr(u.GTIN),
						"CreatedAt":  spanner.CommitTimestamp,
					}))
				}
				return txn.BufferWrite(muts)
			}
		}
		// Fallback to in-memory
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.shipUnits == nil {
			s.shipUnits = map[string][]ShipUnit{}
		}
		s.shipUnits[manifestID] = append(s.shipUnits[manifestID], newUnits...)
		return nil
	}, func(txn outbox.TxnBuffer) error {
		return outbox.EmitJSON(ctx, txn, events.AggregateRoute, manifestID, events.TopicMain, map[string]any{
			"type":          "EventShipUnitsGenerated",
			"manifest_id":   manifestID,
			"units_count":   len(newUnits),
			"supplier_id":   s.resolveSupplierScope(ctx),
			"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
	
	if err != nil {
		return 0, err
	}
	
	return len(newUnits), nil
}"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/payload/ship_units.go", "w") as f:
    f.write(content)
