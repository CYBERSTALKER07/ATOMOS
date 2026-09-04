import SwiftUI

struct SupplyRequestFormData {
    let factoryId: String
    let priority: String
    let notes: String
    let useDemandForecast: Bool
    let requestedDeliveryDate: String?
    let items: [CreateWarehouseSupplyRequestItem]
}

struct CreateSupplyRequestSheet: View {
    let onCreate: (SupplyRequestFormData) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var factoryId = ""
    @State private var factoryLocked = false
    @State private var priority = "NORMAL"
    @State private var notes = ""
    @State private var useForecast = true
    @State private var deliveryDate = Date()
    @State private var includeDeliveryDate = false
    @State private var forecast: [DemandForecastProduct] = []
    @State private var forecastLoading = false
    @State private var manualItems: [ManualSupplyLine] = [ManualSupplyLine()]

    var body: some View {
        NavigationStack {
            Form {
                if factoryLocked {
                    LabeledContent("warehouse_portal.supply_requests.new.text.factory_id", value: factoryId)
                    Text("Nearest factory from the engine.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    TextField("warehouse_portal.supply_requests.new.text.factory_id", text: $factoryId)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }

                Toggle("Requested delivery date", isOn: $includeDeliveryDate)
                if includeDeliveryDate {
                    DatePicker("Delivery date", selection: $deliveryDate, displayedComponents: .date)
                }

                Toggle("Use AI demand forecast", isOn: $useForecast)

                Picker("Priority", selection: $priority) {
                    Text("mobile_warehouse.ui.normal").tag("NORMAL")
                    Text("mobile_warehouse.ui.urgent").tag("URGENT")
                    Text("mobile_warehouse.ui.critical").tag("CRITICAL")
                }
                .pickerStyle(.segmented)

                if useForecast {
                    Section("Demand forecast (7-day)") {
                        if forecastLoading {
                            ProgressView()
                        } else if forecast.isEmpty {
                            Text("mobile_warehouse.ui.no_forecast_data_available")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(forecast, id: \.productId) { product in
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(product.productName.isEmpty ? String(product.productId.prefix(8)) : product.productName)
                                        .font(.subheadline.weight(.semibold))
                                    Text(L10n.format("mobile_warehouse.ui.stock_currentstock_recommended_recommendedqty", "\(product.currentStock)", "\(product.recommendedQty)"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                } else {
                    Section("Manual items") {
                        ForEach($manualItems) { $line in
                            HStack(spacing: LabTheme.spacingSM) {
                                TextField("factory_portal.analytics.text.product_id", text: $line.productId)
                                    .textInputAutocapitalization(.never)
                                TextField("retailer_desktop.pos.text.qty", text: $line.quantity)
                                    .keyboardType(.numberPad)
                                    .frame(width: 72)
                            }
                        }
                        Button("mobile_warehouse.ui.add_item", systemImage: "plus") {
                            manualItems.append(ManualSupplyLine())
                        }
                    }
                }

                TextField("factory_portal.transfers._id_.text.notes", text: $notes, axis: .vertical)
                    .lineLimit(3...5)
            }
            .navigationTitle("warehouse_portal.supply_requests.new.text.new_supply_request")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("warehouse_portal.cycle_counts.text.submit") { submit() }
                        .disabled(!canSubmit)
                }
            }
            .task { await loadEngineFactory() }
            .task(id: useForecast) {
                if useForecast { await loadForecast() }
            }
        }
    }

    private var canSubmit: Bool {
        guard !factoryId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return false }
        if useForecast { return true }
        return manualItems.contains { line in
            !line.productId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            (Int(line.quantity) ?? 0) > 0
        }
    }

    private func loadEngineFactory() async {
        do {
            let supply = try await WarehouseService.opsSupplyFactory()
            if !supply.factoryId.isEmpty {
                factoryId = supply.factoryId
                factoryLocked = true
            }
        } catch {
            factoryLocked = false
        }
    }

    private func loadForecast() async {
        forecastLoading = true
        defer { forecastLoading = false }
        do {
            let response = try await WarehouseService.demandForecast(days: 7)
            forecast = response.products
        } catch {
            forecast = []
        }
    }

    private func submit() {
        let deliveryISO: String? = includeDeliveryDate
            ? ISO8601DateFormatter().string(from: deliveryDate)
            : nil
        let items: [CreateWarehouseSupplyRequestItem]
        if useForecast {
            items = []
        } else {
            items = manualItems.compactMap { line in
                let pid = line.productId.trimmingCharacters(in: .whitespacesAndNewlines)
                guard let qty = Int(line.quantity), !pid.isEmpty, qty > 0 else { return nil }
                return CreateWarehouseSupplyRequestItem(
                    productId: pid,
                    requestedQuantity: qty,
                    recommendedQty: qty,
                    unitVolumeVu: 0
                )
            }
        }
        onCreate(
            SupplyRequestFormData(
                factoryId: factoryId.trimmingCharacters(in: .whitespacesAndNewlines),
                priority: priority,
                notes: notes.trimmingCharacters(in: .whitespacesAndNewlines),
                useDemandForecast: useForecast,
                requestedDeliveryDate: deliveryISO,
                items: items
            )
        )
    }
}

private struct ManualSupplyLine: Identifiable {
    let id = UUID()
    var productId = ""
    var quantity = ""
}
