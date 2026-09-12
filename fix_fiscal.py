import sys

with open('pegasusX/apps/backend-go/order/fiscal.go', 'r') as f:
    content = f.read()

content = content.replace('''	for _, line := range orderRecord.LineItems {
		net := line.Quantity * line.UnitPrice
		vat := (net * vatRateBps) / 10000''', '''	var actualVat int64 = 0
	if len(regime.VatRatesBps) > 0 {
		actualVat = regime.VatRatesBps[0]
	}
	for _, line := range orderRecord.LineItems {
		net := line.Quantity * line.UnitPrice
		vat := (net * actualVat) / 10000''')

content = content.replace('''VatRateBps:  vatRateBps,''', '''VatRateBps:  actualVat,''')

with open('pegasusX/apps/backend-go/order/fiscal.go', 'w') as f:
    f.write(content)
