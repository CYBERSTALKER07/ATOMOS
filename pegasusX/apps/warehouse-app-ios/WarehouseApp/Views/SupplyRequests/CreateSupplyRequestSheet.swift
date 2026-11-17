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
    @State private var vm = CreateSupplyRequestViewModel()

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                if vm.factoryLocked {
                    LabeledContent("warehouse_portal.supply_requests.new.text.factory_id", value: vm.factoryId)
                    Text("Nearest factory from the engine.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    TextField("warehouse_portal.supply_requests.new.text.factory_id", text: $vm.factoryId)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }

                Toggle("Requested delivery date", isOn: $vm.includeDeliveryDate)
                    .onChange(of: vm.includeDeliveryDate) { _, _ in
                        PegasusHaptics.selection()
                    }
                if vm.includeDeliveryDate {
                    DatePicker("Delivery date", selection: $vm.deliveryDate, displayedComponents: .date)
                }

                Toggle("Use AI demand forecast", isOn: $vm.useForecast)
                    .onChange(of: vm.useForecast) { _, _ in
                        PegasusHaptics.selection()
                    }

                Picker("Priority", selection: $vm.priority) {
                    Text("mobile_warehouse.ui.normal").tag("NORMAL")
                    Text("mobile_warehouse.ui.urgent").tag("URGENT")
                    Text("mobile_warehouse.ui.critical").tag("CRITICAL")
                }
                .pickerStyle(.segmented)
                .onChange(of: vm.priority) { _, _ in
                    PegasusHaptics.selection()
                }

                if vm.useForecast {
                    Section("Demand forecast (7-day)") {
                        if vm.forecastLoading {
                            ProgressView()
                        } else if vm.forecast.isEmpty {
                            Text("mobile_warehouse.ui.no_forecast_data_available")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(vm.forecast, id: \.productId) { product in
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(product.productName.isEmpty ? String(product.productId.prefix(8)) : product.productName)
                                        .font(.subheadline.weight(.semibold))
                                    Text(L10n.format("mobile_warehouse.ui.stock_currentstock_recommended_recommendedqty", "\(product.currentStock)", "\(product.recommendedQty)"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .frame(minHeight: 44)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("\(product.productName.isEmpty ? product.productId : product.productName), current stock \(product.currentStock), recommended \(product.recommendedQty)")
                            }
                        }
                    }
                } else {
                    Section("Manual items") {
                        ForEach($vm.manualItems) { $line in
                            HStack(spacing: LabTheme.spacingSM) {
                                TextField("factory_portal.analytics.text.product_id", text: $line.productId)
                                    .textInputAutocapitalization(.never)
                                TextField("retailer_desktop.pos.text.qty", text: $line.quantity)
                                    .keyboardType(.numberPad)
                                    .frame(width: 72)
                            }
                        }
                        Button("mobile_warehouse.ui.add_item", systemImage: "plus") {
                            PegasusHaptics.light()
                            vm.addManualLine()
                        }
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                    }
                }

                TextField("factory_portal.transfers._id_.text.notes", text: $vm.notes, axis: .vertical)
                    .lineLimit(3...5)
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("warehouse_portal.supply_requests.new.text.new_supply_request")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("warehouse_portal.cycle_counts.text.submit") {
                        PegasusHaptics.light()
                        onCreate(vm.buildFormData())
                    }
                    .disabled(!vm.canSubmit)
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            }
            .task { await vm.loadEngineFactory() }
            .task(id: vm.useForecast) {
                if vm.useForecast { await vm.loadForecast() }
            }
        }
    }
}
