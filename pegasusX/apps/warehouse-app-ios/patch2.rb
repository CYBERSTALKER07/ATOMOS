content = File.read("WarehouseApp/Views/Orders/OrderDetailView.swift")

old_func = <<~SWIFT
    private func runMutation(_ action: OrderMutationAction) {
        mutating = true
        statusMessage = nil
        Task {
            defer { mutating = false }
            do {
                let response: WarehouseOrderMutationResponse
                switch action {
                case .reject:
                    let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !reason.isEmpty else {
                        statusMessage = "Reason is required to cancel"
                        return
                    }
                    response = try await WarehouseOperationsService.rejectOrder(orderId: orderId, reason: reason)
                case .overflow:
                    response = try await WarehouseOperationsService.overflowOrder(orderId: orderId, reason: reasonInput.isEmpty ? nil : reasonInput)
                }
                statusMessage = "Order updated · \\(response.status)"
                reasonInput = ""
                load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
SWIFT

new_func = <<~SWIFT
    private func runMutation(_ action: OrderMutationAction) {
        mutating = true
        statusMessage = nil
        Task {
            defer { mutating = false }
            do {
                if action == .paymentBypass {
                    let response = try await WarehouseOperationsService.issuePaymentBypass(orderId: orderId)
                    statusMessage = "Token: \\(response.bypassToken ?? "Unknown")"
                    return
                }

                let response: WarehouseOrderMutationResponse
                switch action {
                case .reject:
                    let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard !reason.isEmpty else {
                        statusMessage = "Reason is required to cancel"
                        return
                    }
                    response = try await WarehouseOperationsService.rejectOrder(orderId: orderId, reason: reason)
                case .overflow:
                    response = try await WarehouseOperationsService.overflowOrder(orderId: orderId, reason: reasonInput.isEmpty ? nil : reasonInput)
                case .paymentBypass:
                    fatalError("Handled above")
                }
                statusMessage = "Order updated · \\(response.status)"
                reasonInput = ""
                load()
            } catch {
                statusMessage = error.localizedDescription
            }
        }
    }
SWIFT

content = content.gsub(old_func.strip, new_func.strip)
File.write("WarehouseApp/Views/Orders/OrderDetailView.swift", content)
