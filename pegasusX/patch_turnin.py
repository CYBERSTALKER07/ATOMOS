import re

with open("apps/backend-go/driver/cash_bag.go", "r") as f:
    content = f.read()

pattern1 = re.compile(r'func \(s \*Service\) TurnInCashBag\(ctx context\.Context, driverID string, req CashBagTurnInRequest\) \(\*CashReconciliation, error\) \{')
replacement1 = r'func (s *Service) TurnInCashBag(ctx context.Context, supplierID, driverID string, req CashBagTurnInRequest) (*CashReconciliation, error) {'
content = content.replace(pattern1.pattern, replacement1)
content = re.sub(r'func \(s \*Service\) TurnInCashBag\(ctx context\.Context, driverID string, req CashBagTurnInRequest\) \(\*CashReconciliation, error\) \{', replacement1, content)


pattern2 = re.compile(r'"ReconciliationId":  reconID,\n\t\t"DriverId":          driverID,\n\t\t"ShiftDate":         shiftDate,')
replacement2 = r'"ReconciliationId":  reconID,\n\t\t"SupplierId":        supplierID,\n\t\t"DriverId":          driverID,\n\t\t"ShiftDate":         shiftDate,'
content = content.replace(pattern2.pattern, replacement2)
content = re.sub(r'"ReconciliationId":  reconID,\n\t\t"DriverId":          driverID,\n\t\t"ShiftDate":         shiftDate,', replacement2, content)


pattern3 = re.compile(r'res, err := s\.TurnInCashBag\(r\.Context\(\), driverID, req\)')
replacement3 = r'''supplierID, _ := auth.ResolveSupplierID(r.Context())
	res, err := s.TurnInCashBag(r.Context(), supplierID, driverID, req)'''
content = content.replace(pattern3.pattern, replacement3)
content = re.sub(r'res, err := s\.TurnInCashBag\(r\.Context\(\), driverID, req\)', replacement3, content)

with open("apps/backend-go/driver/cash_bag.go", "w") as f:
    f.write(content)

