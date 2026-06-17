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
                TextField("Factory ID", text: $factoryId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()

                Toggle("Requested delivery date", isOn: $includeDeliveryDate)
                if includeDeliveryDate {
                    DatePicker("Delivery date", selection: $deliveryDate, displayedComponents: .date)
                }

                Toggle("Use AI demand forecast", isOn: $useForecast)

                Picker("Priority", selection: $priority) {
                    Text("Normal").tag("NORMAL")
                    Text("Urgent").tag("URGENT")
                    Text("Critical").tag("CRITICAL")
                }
                .pickerStyle(.segmented)

                if useForecast {
                    Section("Demand forecast (7-day)") {
                        if forecastLoading {
                            ProgressView()
                        } else if forecast.isEmpty {
                            Text("No forecast data available")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(forecast, id: \.productId) { product in
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(product.productName.isEmpty ? String(product.productId.prefix(8)) : product.productName)
                                        .font(.subheadline.weight(.semibold))
                                    Text("Stock \(product.currentStock) · recommended \(product.recommendedQty)")
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
                                TextField("Product ID", text: $line.productId)
                                    .textInputAutocapitalization(.never)
                                TextField("Qty", text: $line.quantity)
                                    .keyboardType(.numberPad)
                                    .frame(width: 72)
                            }
                        }
                        Button("Add item", systemImage: "plus") {
                            manualItems.append(ManualSupplyLine())
                        }
                    }
                }

                TextField("Notes", text: $notes, axis: .vertical)
                    .lineLimit(3...5)
            }
            .navigationTitle("New Supply Request")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") { submit() }
                        .disabled(!canSubmit)
                }
            }
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
