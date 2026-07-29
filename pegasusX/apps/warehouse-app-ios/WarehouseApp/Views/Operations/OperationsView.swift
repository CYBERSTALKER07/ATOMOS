import SwiftUI

private let broadcastRoles = ["DRIVER", "RETAILER", "ALL"]

struct OperationsView: View {
    @State private var loading = true
    @State private var error: String?
    @State private var statusMessage: String?
    @State private var templates: [BroadcastTemplate] = []

    @State private var title = ""
    @State private var bodyText = ""
    @State private var broadcastRole = "DRIVER"
    @State private var templateDate = ""
    @State private var customReason = ""
    @State private var saveAsTemplate = false
    @State private var broadcasting = false
    @State private var savingTemplate = false

    @State private var productId = ""
    @State private var retailerId = ""
    @State private var proposedPrice = ""
    @State private var preview: RetailerOverridePreview?
    @State private var previewLoading = false
    @State private var previewTask: Task<Void, Never>?

    var body: some View {
        NavigationStack {
            Group {
                if loading && templates.isEmpty && error == nil {
                    WarehouseLoadingView(
                        title: "Loading operations",
                        message: "Fetching broadcast templates and depot tools."
                    )
                } else if let error, templates.isEmpty {
                    WarehouseErrorView(message: error) {
                        Task { await loadTemplates() }
                    }
                } else {
                    operationsForm
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Depot operations")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await loadTemplates(silent: true) }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await loadTemplates() }
            .refreshable { await loadTemplates(silent: true) }
        }
    }

    private var operationsForm: some View {
        Form {
            if let statusMessage {
                Section {
                    Text(statusMessage)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
            }

            OperationsBroadcastForm(
                broadcastRoles: broadcastRoles,
                templates: templates,
                templateDate: $templateDate,
                customReason: $customReason,
                title: $title,
                broadcastRole: $broadcastRole,
                bodyText: $bodyText,
                saveAsTemplate: $saveAsTemplate,
                broadcasting: broadcasting,
                savingTemplate: savingTemplate,
                onApplyTemplate: applyTemplate,
                onDeleteTemplate: deleteTemplate,
                onSendBroadcast: sendBroadcast
            )

            OperationsPricingPreview(
                productId: $productId,
                retailerId: $retailerId,
                proposedPrice: $proposedPrice,
                previewLoading: previewLoading,
                preview: preview,
                onSchedulePreview: schedulePreview
            )
        }
    }

    @MainActor
    private func loadTemplates(silent: Bool = false) async {
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

    private func applyTemplate(_ template: BroadcastTemplate) {
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

    @MainActor
    private func sendBroadcast() async {
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

    @MainActor
    private func deleteTemplate(_ template: BroadcastTemplate) async {
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

    private func schedulePreview() {
        previewTask?.cancel()
        previewTask = Task {
            let product = productId.trimmingCharacters(in: .whitespacesAndNewlines)
            guard let price = Int64(proposedPrice), price > 0, !product.isEmpty else {
                await MainActor.run { preview = nil }
                return
            }
            try? await Task.sleep(nanoseconds: 400_000_000)
            guard !Task.isCancelled else { return }
            await MainActor.run { previewLoading = true }
            defer { Task { @MainActor in previewLoading = false } }
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
                await MainActor.run { preview = result }
            } catch {
                guard !Task.isCancelled else { return }
                await MainActor.run { preview = nil }
            }
        }
    }
}
