import re

with open("apps/backend-go/creditnote/service.go", "r") as f:
    content = f.read()

content = content.replace('line.LineNetMinor = (base.LineNetMinor / base.Qty) * qty', 'line.LineNetMinor = (base.LineNetMinor * qty) / base.Qty')
content = content.replace('line.LineVatMinor = (base.LineVatMinor / base.Qty) * qty', 'line.LineVatMinor = (base.LineVatMinor * qty) / base.Qty')
content = content.replace('line.LineGrossMinor = (base.LineGrossMinor / base.Qty) * qty', 'line.LineGrossMinor = (base.LineGrossMinor * qty) / base.Qty')

with open("apps/backend-go/creditnote/service.go", "w") as f:
    f.write(content)
