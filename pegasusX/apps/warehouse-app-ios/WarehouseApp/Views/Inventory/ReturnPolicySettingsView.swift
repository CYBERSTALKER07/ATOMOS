import SwiftUI

struct ReturnPolicySettingsView: View {
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var saved = false
    @State private var reverseSla = "24"
    @State private var canOverride = false
    @State private var retailerWindow = ""
    @State private var supplierId = ""

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Form {
                    Section {
                        Text("Reverse-dock SLA and optional retailer claim-window override. Override may only lengthen the supplier base window.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }

                    if let error {
                        Section {
                            Text(error).foregroundStyle(.red)
                        }
                    }
                    if saved {
                        Section {
                            Text("Return policy saved.").foregroundStyle(.green)
                        }
                    }

                    Section("Reverse dock") {
                        TextField("SLA hours", text: $reverseSla)
                            .keyboardType(.numberPad)
                            .onChange(of: reverseSla) { _, newValue in
                                let digits = newValue.filter(\.isNumber)
                                if digits != newValue { reverseSla = digits }
                            }
                    }

                    Section("Retailer window") {
                        Toggle("Override retailer claim filing window (lengthen only)", isOn: $canOverride)
                        if canOverride {
                            TextField("Retailer file window (hours)", text: $retailerWindow)
                                .keyboardType(.numberPad)
                                .onChange(of: retailerWindow) { _, newValue in
                                    let digits = newValue.filter(\.isNumber)
                                    if digits != newValue { retailerWindow = digits }
                                }
                        }
                    }

                    Section {
                        Button(saving ? "Saving…" : "Save return policy") { Task { await save() } }
                            .disabled(saving)
                    }
                }
            }
        }
        .navigationTitle("Returns & reverse SLA")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { Task { await load() } }
            }
        }
        .task { await load() }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let wh = TokenStore.shared.warehouseId
            let p = try await WarehouseService.getReturnPolicy(warehouseId: wh)
            supplierId = p.supplierId
            reverseSla = p.reverseDockSlaHours.map(String.init) ?? "24"
            canOverride = p.canOverrideRetailerWindow
            retailerWindow = p.retailerFileWindowHours.map(String.init) ?? ""
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func save() async {
        saving = true
        error = nil
        saved = false
        defer { saving = false }
        var body = WarehouseReturnPolicy(
            supplierId: supplierId,
            reverseDockSlaHours: Int64(reverseSla).flatMap { $0 > 0 ? $0 : nil },
            retailerFileWindowHours: nil,
            canOverrideRetailerWindow: canOverride
        )
        if canOverride {
            guard let hours = Int64(retailerWindow), hours >= 1 else {
                error = "Retailer file window hours required when override is enabled"
                return
            }
            body.retailerFileWindowHours = hours
        }
        do {
            let wh = TokenStore.shared.warehouseId
            _ = try await WarehouseService.putReturnPolicy(body, warehouseId: wh)
            saved = true
        } catch {
            self.error = error.localizedDescription
        }
    }
}
