content = File.read("../supplier-app-ios/SupplierApp/Views/Orders/OrderDetailPanel.swift")

old_str = <<~SWIFT
                    Button("mobile_supplier.ui.reassign_order") {
                        Task { await vm.openReassignDialog(orderId: order.orderId) }
                    }
                    Button("supplier_portal.orders.propose_delay_dialog.text.delay_delivery") { showProposeSheet = true }
                    Button("warehouse_portal.dispatch.text.cancel_order", role: .destructive) { showRejectDialog = true }
SWIFT

new_str = <<~SWIFT
                    Button("mobile_supplier.ui.reassign_order") {
                        Task { await vm.openReassignDialog(orderId: order.orderId) }
                    }
                    Button("supplier_portal.orders.propose_delay_dialog.text.delay_delivery") { showProposeSheet = true }
                    Button("Emergency Payment Bypass") { 
                        Task { await vm.issuePaymentBypass(orderId: order.orderId) }
                    }
                    Button("warehouse_portal.dispatch.text.cancel_order", role: .destructive) { showRejectDialog = true }
SWIFT

content = content.gsub(old_str.strip, new_str.strip)
File.write("../supplier-app-ios/SupplierApp/Views/Orders/OrderDetailPanel.swift", content)
