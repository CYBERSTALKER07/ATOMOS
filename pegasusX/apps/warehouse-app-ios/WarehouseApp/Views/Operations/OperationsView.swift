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

            Section {
                WarehouseSectionHeader(
                    title: "Broadcast templates",
                    subtitle: "Built-in depot starters plus your saved custom messages."
                )
            }
            Section {
                if templates.isEmpty {
                    Text("No templates available.")
                        .foregroundStyle(.secondary)
                } else {
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: LabTheme.spacingSM) {
                            ForEach(templates) { template in
                                HStack(spacing: 4) {
                                    Button {
                                        applyTemplate(template)
                                    } label: {
                                        let suffix = template.source == "custom" ? " · saved" : ""
                                        Text(template.title + suffix)
                                    }
                                    .buttonStyle(.bordered)
                                    .font(.caption)

                                    if template.source == "custom" {
                                        Button(role: .destructive) {
                                            Task { await deleteTemplate(template) }
                                        } label: {
                                            Image(systemName: "xmark.circle.fill")
                                        }
                                        .buttonStyle(.borderless)
                                        .font(.caption)
                                    }
                                }
                            }
                        }
                    }
                }
            }

            Section {
                WarehouseSectionHeader(title: "Send depot broadcast")
            }
            Section {
                TextField("Effective date (optional)", text: $templateDate)
                    .textInputAutocapitalization(.never)
                TextField("Custom reason (optional)", text: $customReason)
                TextField("Title", text: $title)
                Picker("Target role", selection: $broadcastRole) {
                    ForEach(broadcastRoles, id: \.self) { role in
                        Text(role).tag(role)
                    }
                }
                TextField("Message", text: $bodyText, axis: .vertical)
                    .lineLimit(4...8)
                Toggle("Save as custom template for this depot", isOn: $saveAsTemplate)
                Button(broadcasting || savingTemplate ? "Sending…" : "Send broadcast") {
                    Task { await sendBroadcast() }
                }
                .disabled(
                    broadcasting || savingTemplate
                        || title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        || bodyText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                )
            }

            Section {
                WarehouseSectionHeader(
                    title: "Pricing impact preview (read-only)",
                    subtitle: "Preview proposed retailer price vs catalog list price. Does not create overrides."
                )
            }
            Section {
                TextField("Product / SKU ID", text: $productId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onChange(of: productId) { _, _ in schedulePreview() }
                TextField("Retailer ID (optional)", text: $retailerId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .onChange(of: retailerId) { _, _ in schedulePreview() }
                TextField("Proposed price (minor units)", text: $proposedPrice)
                    .keyboardType(.numberPad)
                    .onChange(of: proposedPrice) { _, _ in schedulePreview() }

                if previewLoading {
                    Text("Loading preview…")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                if let preview {
                    LabeledContent("Retailers on SKU", value: "\(preview.retailersOnSkuCount)")
                    LabeledContent("Active overrides", value: "\(preview.activeOverrideCount)")
                    LabeledContent("Catalog list price", value: "\(preview.catalogListPrice)")
                    LabeledContent("Margin delta / unit", value: "\(preview.marginDeltaPerUnit)")
                    Text(preview.marginEstimateLabel)
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                    if preview.readOnly == true {
                        Text("Read-only — contact supplier to apply overrides.")
                            .font(.footnote.weight(.medium))
                    }
                }
            }
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
