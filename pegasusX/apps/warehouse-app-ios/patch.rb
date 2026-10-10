content = File.read("WarehouseApp/Views/Orders/OrderDetailView.swift")

content = content.gsub("case .overflow: return \"Return to dispatch pool?\"\n        }", "case .overflow: return \"Return to dispatch pool?\"\n        case .paymentBypass: return \"Issue Payment Bypass?\"\n        }")

content = content.gsub("case .overflow: return \"Optional reason can be entered above.\"\n        }", "case .overflow: return \"Optional reason can be entered above.\"\n        case .paymentBypass: return \"Generate a token for the driver if the POS terminal failed.\"\n        }")

content = content.gsub("case .overflow: return \"Return to pool\"\n        }", "case .overflow: return \"Return to pool\"\n        case .paymentBypass: return \"Issue Bypass Token\"\n        }")

File.write("WarehouseApp/Views/Orders/OrderDetailView.swift", content)
