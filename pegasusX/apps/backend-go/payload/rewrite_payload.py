import re

# Fix apply.go
with open("apply.go", "r") as f:
    apply_code = f.read()

apply_code = apply_code.replace("s.rebuildIndexesLocked()", """s.manifestOrders = make(map[string][]ManifestOrder)
		s.overflowCount = make(map[string]int64)
		for _, o := range s.orders {
			if o.ManifestID != "" {
				s.manifestOrders[o.ManifestID] = append(s.manifestOrders[o.ManifestID], ManifestOrder{
					OrderID: o.OrderID,
					TotalVU: o.TotalVU,
					State: o.State,
					OrderType: o.OrderType,
					StopIndex: o.StopIndex,
					OriginalTotalVU: o.OriginalTotalVU,
				})
			}
		}""")

with open("apply.go", "w") as f:
    f.write(apply_code)


# Fix repository_spanner.go
with open("repository_spanner.go", "r") as f:
    spanner_code = f.read()

spanner_code = spanner_code.replace("s.rebuildIndexesLocked()", """s.manifestOrders = make(map[string][]ManifestOrder)
		s.overflowCount = make(map[string]int64)
		for _, o := range s.orders {
			if o.ManifestID != "" {
				s.manifestOrders[o.ManifestID] = append(s.manifestOrders[o.ManifestID], ManifestOrder{
					OrderID: o.OrderID,
					TotalVU: o.TotalVU,
					State: o.State,
					OrderType: o.OrderType,
					StopIndex: o.StopIndex,
					OriginalTotalVU: o.OriginalTotalVU,
				})
			}
		}""")

# Fix fields for ManifestRow
spanner_code = spanner_code.replace("m.TransferCnt", "m.StopCount")
spanner_code = re.sub(r'm\.DispatchedAt.*?\}', '', spanner_code)
spanner_code = re.sub(r'm\.CompletedAt.*?\}', '', spanner_code)
spanner_code = re.sub(r'm\.CancelledAt.*?\}', '', spanner_code)
spanner_code = re.sub(r'DispatchedAt.*?CancelledAt.*?\n', '', spanner_code)
spanner_code = re.sub(r'&dispatchedAt,\s*&completedAt,\s*&cancelledAt', '', spanner_code)

with open("repository_spanner.go", "w") as f:
    f.write(spanner_code)

print("Updated payload files")
