content = File.read("WarehouseApp/Components/Orders/OrderOpsActions.swift")

content = content.gsub("[\"COMPLETED\", \"CANCELLED\"].include?(state.uppercased())", "[\"COMPLETED\", \"CANCELLED\"].contains(state.uppercased())")

File.write("WarehouseApp/Components/Orders/OrderOpsActions.swift", content)
