import SwiftUI

struct OrgFleetView: View {
    @State private var viewModel = OrgFleetViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.topology == nil && viewModel.drivers.isEmpty && viewModel.vehicles.isEmpty && viewModel.orgMembers.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.org_fleet")
            } else if let error = viewModel.error, viewModel.topology == nil && viewModel.drivers.isEmpty && viewModel.vehicles.isEmpty && viewModel.orgMembers.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.reload(silent: true) } }
            } else {
                VStack(spacing: 0) {
                    MetricsOverviewView(
                        driversCount: viewModel.drivers.count,
                        vehiclesCount: viewModel.vehicles.count,
                        orgMembersCount: viewModel.orgMembers.count,
                        topology: viewModel.topology
                    )
                    .padding(.top)

                    Picker("mobile_supplier.chrome.section", selection: $vm.tab) {
                        Text(L10n.format("mobile_supplier.ui.drivers_count", "\(viewModel.drivers.count)")).tag(0)
                        Text(L10n.format("mobile_supplier.ui.vehicles_count", "\(viewModel.vehicles.count)")).tag(1)
                        Text(L10n.format("mobile_supplier.ui.org_count", "\(viewModel.orgMembers.count)")).tag(2)
                    }
                    .pickerStyle(.segmented)
                    .padding()

                    switch viewModel.tab {
                    case 0:
                        DriverListView(drivers: viewModel.drivers, topology: viewModel.topology)
                    case 1:
                        VehicleListView(vehicles: viewModel.vehicles, topology: viewModel.topology)
                    default:
                        OrgMemberListView(
                            orgMembers: viewModel.orgMembers,
                            editingMember: $vm.editingMember,
                            showEditMemberSheet: $vm.showEditMemberSheet,
                            memberActionId: $vm.memberActionId,
                            deactivateAction: { userId in
                                await viewModel.deactivateMember(userId)
                            }
                        )
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("mobile_supplier.ui.org_and_fleet")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("mobile_supplier.ui.add") {
                    viewModel.openCreateForCurrentTab()
                }
            }
        }
        .task { await viewModel.reload() }
        .sheet(isPresented: $vm.showDriverSheet) {
            if let topology = viewModel.topology {
                CreateDriverSheet(topology: topology, vehicles: viewModel.vehicles) {
                    viewModel.showDriverSheet = false
                    Task { await viewModel.reload(silent: true) }
                }
            }
        }
        .sheet(isPresented: $vm.showVehicleSheet) {
            if let topology = viewModel.topology {
                CreateVehicleSheet(topology: topology) {
                    viewModel.showVehicleSheet = false
                    Task { await viewModel.reload(silent: true) }
                }
            }
        }
        .sheet(isPresented: $vm.showOrgSheet) {
            if let topology = viewModel.topology {
                CreateOrgMemberSheet(topology: topology) {
                    viewModel.showOrgSheet = false
                    Task { await viewModel.reload(silent: true) }
                }
            }
        }
        .sheet(isPresented: $vm.showEditMemberSheet) {
            if let editingMember = viewModel.editingMember {
                EditOrgMemberSheet(member: editingMember) {
                    viewModel.showEditMemberSheet = false
                    Task { await viewModel.reload(silent: true) }
                }
            }
        }
    }
}
