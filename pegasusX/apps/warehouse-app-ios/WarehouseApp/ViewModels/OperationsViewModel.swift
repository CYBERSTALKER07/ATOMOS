import Foundation
import Observation

@Observable
@MainActor
final class OperationsViewModel {
    let broadcastRoles = ["DRIVER", "RETAILER", "ALL"]

    var loading = true
    var error: String?
    var statusMessage: String?
    var templates: [BroadcastTemplate] = []

    var title = ""
    var bodyText = ""
    var broadcastRole = "DRIVER"
    var templateDate = ""
    var customReason = ""
    var saveAsTemplate = false
    var broadcasting = false
    var savingTemplate = false

    var productId = ""
    var retailerId = ""
    var proposedPrice = ""
    var preview: RetailerOverridePreview?
    var previewLoading = false
    private var previewTask: Task<Void, Never>?

    init() {}

    func loadTemplates(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let response = try await WarehouseService.broadcastTemplates()
            templates = response.templates
        } catch {
            if !silent || templates.isEmpty {
                self.error = error.localizedDescription
            }
        }
    }

    func applyTemplate(_ template: BroadcastTemplate) {
        var resolvedBody = template.body
        if resolvedBody.contains("{date}") {
            let date = templateDate.trimmingCharacters(in: .whitespacesAndNewlines)
            resolvedBody = resolvedBody.replacingOccurrences(
                of: "{date}",
                with: date.isEmpty ? "the selected date" : date
            )
        }
        if resolvedBody.contains("{reason}") {
            let reason = customReason.trimmingCharacters(in: .whitespacesAndNewlines)
            resolvedBody = resolvedBody.replacingOccurrences(
                of: "{reason}",
                with: reason.isEmpty ? "operational delay" : reason
            )
        }
        title = template.title
        bodyText = resolvedBody
        if broadcastRoles.contains(template.defaultRole) {
            broadcastRole = template.defaultRole
        }
    }

    func sendBroadcast() async {
        let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedBody = bodyText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedTitle.isEmpty, !trimmedBody.isEmpty else { return }

        broadcasting = true
        statusMessage = nil
        defer {
            broadcasting = false
            savingTemplate = false
        }

        do {
            if saveAsTemplate {
                savingTemplate = true
                let createKey = WarehouseIdempotency.broadcastTemplateCreate(title: trimmedTitle, body: trimmedBody)
                _ = try await WarehouseService.createBroadcastTemplate(
                    WarehouseBroadcastTemplateCreateRequest(
                        title: trimmedTitle,
                        body: trimmedBody,
                        defaultRole: broadcastRole,
                        category: "custom"
                    ),
                    idempotencyKey: createKey
                )
            }
            let broadcastKey = WarehouseIdempotency.broadcast(
                role: broadcastRole,
                title: trimmedTitle,
                body: trimmedBody
            )
            let response = try await WarehouseService.postBroadcast(
                WarehouseBroadcastRequest(title: trimmedTitle, body: trimmedBody, role: broadcastRole),
                idempotencyKey: broadcastKey
            )
            statusMessage = "Broadcast sent from depot \(response.warehouseId)."
            title = ""
            bodyText = ""
            saveAsTemplate = false
            await loadTemplates(silent: true)
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    func deleteTemplate(_ template: BroadcastTemplate) async {
        guard template.source == "custom" else { return }
        statusMessage = nil
        do {
            let key = WarehouseIdempotency.broadcastTemplateDelete(templateId: template.id)
            _ = try await WarehouseService.deleteBroadcastTemplate(templateId: template.id, idempotencyKey: key)
            statusMessage = "Custom template removed."
            await loadTemplates(silent: true)
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    func schedulePreview() {
        previewTask?.cancel()
        previewTask = Task {
            let product = productId.trimmingCharacters(in: .whitespacesAndNewlines)
            guard let price = Int64(proposedPrice), price > 0, !product.isEmpty else {
                preview = nil
                return
            }
            try? await Task.sleep(nanoseconds: 400_000_000)
            guard !Task.isCancelled else { return }
            previewLoading = true
            defer { previewLoading = false }
            do {
                let result = try await WarehouseService.previewRetailerPriceOverride(
                    RetailerOverridePreviewRequest(
                        retailerId: retailerId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                            ? nil : retailerId.trimmingCharacters(in: .whitespacesAndNewlines),
                        productId: product,
                        skuId: nil,
                        proposedPrice: price
                    )
                )
                guard !Task.isCancelled else { return }
                preview = result
            } catch {
                guard !Task.isCancelled else { return }
                preview = nil
            }
        }
    }
}
