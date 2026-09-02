content = File.read("../supplier-app-ios/SupplierApp/ViewModels/OrdersViewModel.swift")

bypass_func = <<~SWIFT
    func issuePaymentBypass(orderId: String) async {
        mutating = true
        defer { mutating = false }
        do {
            let req = PaymentBypassRequest(orderId: orderId)
            let idempotency = UUID().uuidString
            let res = try await SupplierOperationsService.issuePaymentBypass(req, idempotencyKey: idempotency)
            if let token = res.bypassToken {
                opsError = "Token generated: \\(token)"
            } else {
                opsError = "Bypass request successful, but no token returned."
            }
        } catch {
            opsError = error.localizedDescription
        }
    }
SWIFT

content = content.sub("    func closeReassignDialog() {", bypass_func + "\n    func closeReassignDialog() {")
File.write("../supplier-app-ios/SupplierApp/ViewModels/OrdersViewModel.swift", content)
