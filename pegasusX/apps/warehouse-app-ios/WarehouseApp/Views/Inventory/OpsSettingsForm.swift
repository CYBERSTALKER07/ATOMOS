import SwiftUI

struct FeeTierDraft: Identifiable, Equatable {
    let id = UUID()
    var maxKm: String
    var feeMinor: String

    init(maxKm: String = "", feeMinor: String = "0") {
        self.maxKm = maxKm
        self.feeMinor = feeMinor
    }
}

struct OpsSettingsForm: View {
    @Binding var preorderMinLeadDays: String
    @Binding var preorderMaxLeadDays: String
    @Binding var policy: String
    @Binding var showStockCounts: Bool
    @Binding var clearOrderLineMin: Bool
    @Binding var orderLineMin: String
    @Binding var clearOrderLineMax: Bool
    @Binding var orderLineMax: String
    @Binding var expressEnabled: Bool
    @Binding var expressStockFloor: String
    @Binding var clearFeeRules: Bool
    @Binding var feeBaseMinor: String
    @Binding var feeCurrency: String
    @Binding var feeTiers: [FeeTierDraft]
    @Binding var enforceOrderAcceptance: Bool
    @Binding var scheduleIs24h: Bool
    @Binding var scheduleTimezone: String
    @Binding var weekdayOpen: String
    @Binding var weekdayClose: String
    @Binding var scheduleJSON: String
    let scheduleError: String?

    var body: some View {
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

        Section("Order acceptance hours") {
            Text("When enforcement is on, retailers cannot preview or create orders outside the window.")
                .font(.caption)
                .foregroundStyle(.secondary)
            Toggle("Enforce order acceptance hours", isOn: $enforceOrderAcceptance)
            Toggle("Open 24 hours", isOn: $scheduleIs24h)
            TextField("Timezone", text: $scheduleTimezone)
            HStack {
                TextField("Weekday open", text: $weekdayOpen)
                TextField("Weekday close", text: $weekdayClose)
            }
            Text("Advanced JSON").font(.caption).foregroundStyle(.secondary)
            TextEditor(text: $scheduleJSON)
                .font(.system(.caption, design: .monospaced))
                .frame(minHeight: 140)
            if let scheduleError {
                Text(scheduleError).foregroundStyle(.red).font(.caption)
            }
        }
    }
}
