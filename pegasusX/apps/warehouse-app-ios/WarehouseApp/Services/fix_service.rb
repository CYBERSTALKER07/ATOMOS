content = File.read('WarehouseOperationsService.swift')

# Replace the incorrect part
new_content = content.sub("    }\nimport Foundation", "    }\n}\n\n// Models\n")
File.write('WarehouseOperationsService.swift', new_content)
