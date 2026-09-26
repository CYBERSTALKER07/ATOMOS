import re

with open("apps/backend-go/driver/cash_bag.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(s \*Service\) HandleListCashReconciliations\(w http\.ResponseWriter, r \*http\.Request\) \{\n\tif s\.spanner == nil \{\n\t\twriteJSON\(w, http\.StatusOK, map\[string\]any\{"reconciliations": \[\]CashReconciliation\{\}\}\)\n\t\treturn\n\t\}\n\tstatus := strings\.TrimSpace\(r\.URL\.Query\(\)\.Get\("status"\)\)\n\n\tsql := `SELECT ReconciliationId, DriverId, RouteId, ShiftDate, ExpectedCashMinor,\n\t               DeclaredCashMinor, DifferenceMinor, Status, DriverNote, FinanceNote,\n\t               CreatedAt, ResolvedAt, ResolvedBy\n\t        FROM CashReconciliations WHERE 1=1`\n\tparams := map\[string\]any\{\}')

replacement = r"""func (s *Service) HandleListCashReconciliations(w http.ResponseWriter, r *http.Request) {
	if s.spanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"reconciliations": []CashReconciliation{}})
		return
	}
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing_supplier_scope"})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	sql := `SELECT ReconciliationId, DriverId, RouteId, ShiftDate, ExpectedCashMinor,
	               DeclaredCashMinor, DifferenceMinor, Status, DriverNote, FinanceNote,
	               CreatedAt, ResolvedAt, ResolvedBy
	        FROM CashReconciliations WHERE SupplierId = @supplierId`
	params := map[string]any{"supplierId": supplierID}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/driver/cash_bag.go", "w") as f:
    f.write(content)
