content = File.read("WarehouseApp/Components/Orders/OrderOpsActions.swift")

content = content.gsub("                if canReassign(state) {\n                    Button(\"mobile_warehouse.ui.reassign_order\") { onLoadRecommendations() }\n                        .disabled(mutating)\n                }", "                if canReassign(state) {\n                    Button(\"mobile_warehouse.ui.reassign_order\") { onLoadRecommendations() }\n                        .disabled(mutating)\n                }\n                if ![\"COMPLETED\", \"CANCELLED\"].include?(state.uppercased()) {\n                    Button(\"Emergency Payment Bypass\") { pendingAction = .paymentBypass }\n                        .disabled(mutating)\n                }")

File.write("WarehouseApp/Components/Orders/OrderOpsActions.swift", content)
