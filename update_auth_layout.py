import re

path = '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-portal/app/auth/layout.tsx'
with open(path, 'r') as f:
    content = f.read()

content = content.replace("Pegasus &copy; 2026", "pegasusX &copy; 2026")
content = content.replace('alt="Pegasus"', 'alt="pegasusX Supplier"')

with open(path, 'w') as f:
    f.write(content)
