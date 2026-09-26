import re

with open("apps/backend-go/retailer/store_stock.go", "r") as f:
    content = f.read()

replacement = """		if s.spannerClient == nil {
			for _, l := range lines {
				if l.Variance == 0 {
					continue
				}
				if err := s.applyAdjust(r.Context(), orgID, locID, bin, l.Sku, l.Variance, actor, "cycle_count:"+countID); err != nil {
					writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "sku": l.Sku})
					return
				}
			}
		} else {
			_, err := s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				for _, l := range lines {
					if l.Variance == 0 {
						continue
					}
					if err := s.applyDeltaInTxn(ctx, txn, orgID, locID, bin, l.Sku, l.Variance, MoveCountVariance, "ADJUST", "", actor, "cycle_count:"+countID); err != nil {
						return fmt.Errorf("sku %s: %w", l.Sku, err)
					}
				}
				return nil
			})
			if err != nil {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
		}

		for _, l := range lines {
			if l.Variance != 0 {
				_ = s.syncReorderCurrentStock(r.Context(), orgID, l.Sku)
			}
		}"""

pattern = re.compile(r'		for _, l := range lines \{\n\t\t\tif l\.Variance == 0 \{\n\t\t\t\tcontinue\n\t\t\t\}\n\t\t\tif err := s\.applyAdjust\(r\.Context\(\), orgID, locID, bin, l\.Sku, l\.Variance, actor, "cycle_count:"\+countID\); err != nil \{\n\t\t\t\twriteJSON\(w, http\.StatusConflict, map\[string\]string\{"error": err\.Error\(\), "sku": l\.Sku\}\)\n\t\t\t\treturn\n\t\t\t\}\n\t\t\t_ = s\.syncReorderCurrentStock\(r\.Context\(\), orgID, l\.Sku\)\n\t\t\}', re.DOTALL)
content = pattern.sub(replacement, content)

with open("apps/backend-go/retailer/store_stock.go", "w") as f:
    f.write(content)
