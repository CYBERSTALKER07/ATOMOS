import SwiftUI

private struct FeeTierDraft: Identifiable, Equatable {
    let id = UUID()
    var maxKm: String
    var feeMinor: String

    init(maxKm: String = "", feeMinor: String = "0") {
        self.maxKm = maxKm
        self.feeMinor = feeMinor
    }
}

struct OpsSettingsView: View {
    @State private var policy = "REJECT"
    @State private var showStockCounts = false
    @State private var preorderMinLeadDays = "3"
    @State private var preorderMaxLeadDays = "90"
    @State private var orderLineMin = ""
    @State private var orderLineMax = ""
    @State private var clearOrderLineMin = true
    @State private var clearOrderLineMax = true
    @State private var expressEnabled = false
    @State private var expressStockFloor = "0"
    @State private var feeBaseMinor = "0"
    @State private var feeCurrency = "UZS"
    @State private var feeTiers: [FeeTierDraft] = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
    @State private var clearFeeRules = true
    @State private var scheduleJSON = "{\n  \"is_24h\": true\n}"
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var scheduleError: String?
    @State private var saveMessage: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { load() }
                }
            } else {
                Form {
                    Section {
                        Text("Checkout policy, pre-orders, delivery fees, and retailer catalog display.")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }

                    Section("Pre-order lead window") {
                        Text("Retailers can request delivery between these lead days from today.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        HStack {
                            TextField("Min days", text: $preorderMinLeadDays)
                                .keyboardType(.numberPad)
                            TextField("Max days", text: $preorderMaxLeadDays)
                                .keyboardType(.numberPad)
                        }
                    }

                    Section("Out-of-stock orders") {
                        Toggle("Accept when out of stock", isOn: Binding(
                            get: { policy == "ACCEPT_BACKORDER" },
                            set: { policy = $0 ? "ACCEPT_BACKORDER" : "REJECT" }
                        ))
                        Picker("Policy", selection: $policy) {
                            Text("Reject").tag("REJECT")
                            Text("Accept backorder").tag("ACCEPT_BACKORDER")
                        }
                        .pickerStyle(.inline)
                    }

                    Section("Retailer catalog display") {
                        Toggle("Show stock counts to retailers", isOn: $showStockCounts)
                    }

                    Section("Order line quantity limits") {
                        Toggle("No minimum quantity", isOn: $clearOrderLineMin)
                        if !clearOrderLineMin {
                            TextField("Minimum quantity", text: $orderLineMin)
                                .keyboardType(.numberPad)
                        }
                        Toggle("No maximum quantity", isOn: $clearOrderLineMax)
                        if !clearOrderLineMax {
                            TextField("Maximum quantity", text: $orderLineMax)
                                .keyboardType(.numberPad)
                        }
                    }

                    Section("Express delivery") {
                        Toggle("Express enabled", isOn: $expressEnabled)
                        TextField("Express stock floor", text: $expressStockFloor)
                            .keyboardType(.numberPad)
                    }

                    Section("Delivery fee rules") {
                        Toggle("No delivery fee rules", isOn: $clearFeeRules)
                        if !clearFeeRules {
                            TextField("Base fee (minor)", text: $feeBaseMinor)
                                .keyboardType(.numberPad)
                            TextField("Currency", text: $feeCurrency)
                            ForEach($feeTiers) { $tier in
                                HStack {
                                    TextField("Max km", text: $tier.maxKm)
                                        .keyboardType(.decimalPad)
                                    TextField("Fee (minor)", text: $tier.feeMinor)
                                        .keyboardType(.numberPad)
                                }
                            }
                            Button("Add tier") {
                                feeTiers.append(FeeTierDraft())
                            }
                            if feeTiers.count > 1 {
                                Button("Remove last tier", role: .destructive) {
                                    feeTiers.removeLast()
                                }
                            }
                        }
                    }

                    Section("Operating hours (display only)") {
                        Text("Shown to retailers for planning. Dispatch and delivery are not blocked outside these hours.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        TextEditor(text: $scheduleJSON)
                            .font(.system(.caption, design: .monospaced))
                            .frame(minHeight: 140)
                        if let scheduleError {
                            Text(scheduleError).foregroundStyle(.red).font(.caption)
                        }
                    }

                    if let saveMessage {
                        Section {
                            Text(saveMessage)
                                .foregroundStyle(saveMessage.contains("saved") ? .green : .red)
                        }
                    }

                    Section {
                        Button(saving ? "Saving…" : "Save settings") { save() }
                            .disabled(saving)
                    }
                }
            }
        }
        .navigationTitle("Ops settings")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let settings = try await WarehouseService.opsSettings()
                policy = settings.defaultOutOfStockPolicy == "ACCEPT_BACKORDER" ? "ACCEPT_BACKORDER" : "REJECT"
                showStockCounts = settings.showStockCountsToRetailers
                preorderMinLeadDays = String(settings.preorderMinLeadDays)
                preorderMaxLeadDays = String(settings.preorderMaxLeadDays)
                if let minQty = settings.orderLineMinQuantity {
                    orderLineMin = String(minQty)
                    clearOrderLineMin = false
                } else {
                    orderLineMin = ""
                    clearOrderLineMin = true
                }
                if let maxQty = settings.orderLineMaxQuantity {
                    orderLineMax = String(maxQty)
                    clearOrderLineMax = false
                } else {
                    orderLineMax = ""
                    clearOrderLineMax = true
                }
                expressEnabled = settings.expressEnabled
                expressStockFloor = String(settings.expressStockFloor)
                if let rules = settings.deliveryFeeRules {
                    feeBaseMinor = String(rules.baseFeeMinor)
                    feeCurrency = rules.currency.isEmpty ? "UZS" : rules.currency
                    feeTiers = rules.tiers.map {
                        FeeTierDraft(
                            maxKm: $0.maxKm.map { String($0) } ?? "",
                            feeMinor: String($0.feeMinor)
                        )
                    }
                    if feeTiers.isEmpty {
                        feeTiers = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
                    }
                    clearFeeRules = false
                } else {
                    feeBaseMinor = "0"
                    feeCurrency = "UZS"
                    feeTiers = [FeeTierDraft(maxKm: "5", feeMinor: "0")]
                    clearFeeRules = true
                }
                if let schedule = settings.operatingSchedule {
                    scheduleJSON = schedule.prettyJSONString()
                }
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func save() {
        saving = true
        saveMessage = nil
        scheduleError = nil
        guard let data = scheduleJSON.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            scheduleError = "Operating schedule must be valid JSON"
            saving = false
            return
        }
        guard let minLead = Int64(preorderMinLeadDays),
              let maxLead = Int64(preorderMaxLeadDays) else {
            saveMessage = "Pre-order lead days must be valid numbers"
            saving = false
            return
        }
        let schedule = object.mapValues { AnyCodable($0) }
        let deliveryRules: DeliveryFeeRules? = clearFeeRules ? nil : DeliveryFeeRules(
            currency: feeCurrency.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "UZS" : feeCurrency,
            baseFeeMinor: Int64(feeBaseMinor) ?? 0,
            tiers: feeTiers.map {
                let trimmed = $0.maxKm.trimmingCharacters(in: .whitespacesAndNewlines)
                return DeliveryFeeTier(
                    maxKm: trimmed.isEmpty ? nil : Double(trimmed),
                    feeMinor: Int64($0.feeMinor) ?? 0
                )
            }
        )
        let patch = WarehouseOpsSettingsPatchRequest(
            defaultOutOfStockPolicy: policy,
            showStockCountsToRetailers: showStockCounts,
            operatingSchedule: schedule,
            preorderMinLeadDays: minLead,
            preorderMaxLeadDays: maxLead,
            orderLineMinQuantity: clearOrderLineMin ? nil : Int64(orderLineMin),
            orderLineMaxQuantity: clearOrderLineMax ? nil : Int64(orderLineMax),
            clearOrderLineMinQuantity: clearOrderLineMin ? true : nil,
            clearOrderLineMaxQuantity: clearOrderLineMax ? true : nil,
            expressEnabled: expressEnabled,
            expressStockFloor: Int64(expressStockFloor) ?? 0,
            deliveryFeeRules: deliveryRules,
            clearDeliveryFeeRules: clearFeeRules ? true : nil
        )
        Task {
            do {
                try await WarehouseService.patchOpsSettings(patch)
                saveMessage = "Warehouse settings saved"
                load()
            } catch {
                saveMessage = error.localizedDescription
            }
            saving = false
        }
    }
}
