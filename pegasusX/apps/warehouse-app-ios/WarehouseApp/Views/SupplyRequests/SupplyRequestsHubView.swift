import SwiftUI

struct SupplyRequestsHubView: View {
    @State private var vm = SupplyRequestsHubViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading {
                WarehouseLoadingView(
                    title: "Loading supply requests",
                    message: "Fetching factory supply pipeline status."
                )
            } else if let error = vm.error {
                WarehouseErrorView(message: error) {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
            } else if vm.filtered.isEmpty {
                WarehouseEmptyView(
                    title: "No requests",
                    message: "No supply requests in this state."
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.filtered) { request in
                        NavigationLink {
                            SupplyRequestDetailView(requestId: request.requestId)
                        } label: {
                            HStack(alignment: .top) {
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(String(request.requestId.prefix(8)))
                                        .font(.headline)
                                    Text(L10n.format("mobile_warehouse.ui.priority_totalvolumevu_vu", "\(request.priority)", "\(Int(request.totalVolumeVu))"))
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                    if !request.notes.isEmpty {
                                        Text(request.notes)
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                            .lineLimit(2)
                                    }
                                }
                                Spacer()
                                WarehouseStatusBadge(text: request.state)
                            }
                            .frame(minHeight: 44)
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Request \(String(request.requestId.prefix(8))), \(request.priority), \(Int(request.totalVolumeVu)) VU, \(request.state)")
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("portal.nav.supply_requests")
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Picker("State", selection: $vm.stateFilter) {
                    ForEach(vm.filters, id: \.self) { Text($0).tag($0) }
                }
                .pickerStyle(.menu)
                .frame(minHeight: 44)
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    vm.showCreate = true
                } label: {
                    Label("mobile_warehouse.ui.new", systemImage: "plus")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("New supply request")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh supply requests")
            }
        }
        .sheet(isPresented: $vm.showCreate) {
            CreateSupplyRequestSheet { form in
                Task { await vm.submitCreate(form: form) }
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
        .onChange(of: vm.stateFilter) { _, _ in
            PegasusHaptics.selection()
            Task { await vm.load() }
        }
    }
}
