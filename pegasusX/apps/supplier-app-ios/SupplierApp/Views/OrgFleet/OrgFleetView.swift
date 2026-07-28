import SwiftUI

struct OrgFleetView: View {
    @State private var tab = 0
    @State private var loading = true
    @State private var error: String?
    @State private var topology: SupplierTopologyResponse?
    @State private var drivers: [FleetDriver] = []
    @State private var vehicles: [FleetVehicle] = []
    @State private var orgMembers: [SupplierOrgMember] = []
    @State private var showDriverSheet = false
    @State private var showVehicleSheet = false
    @State private var showOrgSheet = false
    @State private var showEditMemberSheet = false
    @State private var editingMember: SupplierOrgMember?
    @State private var memberActionId: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading org & fleet…")
            } else if let error {
                SupplierErrorView(message: error) { Task { await reload() } }
            } else {
                VStack(spacing: 0) {
                    MetricsOverviewView(
                        driversCount: drivers.count,
                        vehiclesCount: vehicles.count,
                        orgMembersCount: orgMembers.count,
                        topology: topology
                    )
                    .padding(.top)

                    Picker("Section", selection: $tab) {
                        Text("Drivers (\(drivers.count))").tag(0)
                        Text("Vehicles (\(vehicles.count))").tag(1)
                        Text("Org (\(orgMembers.count))").tag(2)
                    }
                    .pickerStyle(.segmented)
                    .padding()

                    switch tab {
                    case 0:
                        DriverListView(drivers: drivers, topology: topology)
                    case 1:
                        VehicleListView(vehicles: vehicles, topology: topology)
                    default:
                        OrgMemberListView(
                            orgMembers: orgMembers,
                            editingMember: $editingMember,
                            showEditMemberSheet: $showEditMemberSheet,
                            memberActionId: $memberActionId,
                            deactivateAction: deactivateMember
                        )
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Org & fleet")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("Add") {
                    switch tab {
                    case 0: showDriverSheet = true
                    case 1: showVehicleSheet = true
                    default: showOrgSheet = true
                    }
                }
            }
        }
        .task { await reload() }
        .sheet(isPresented: $showDriverSheet) {
            if let topology {
                CreateDriverSheet(topology: topology, vehicles: vehicles) {
                    showDriverSheet = false
                    Task { await reload() }
                }
            }
        }
        .sheet(isPresented: $showVehicleSheet) {
            if let topology {
                CreateVehicleSheet(topology: topology) {
                    showVehicleSheet = false
                    Task { await reload() }
                }
            }
        }
        .sheet(isPresented: $showOrgSheet) {
            if let topology {
                CreateOrgMemberSheet(topology: topology) {
                    showOrgSheet = false
                    Task { await reload() }
                }
            }
        }
        .sheet(isPresented: $showEditMemberSheet) {
            if let editingMember {
                EditOrgMemberSheet(member: editingMember) {
                    showEditMemberSheet = false
                    Task { await reload() }
                }
            }
        }
    }

    private func reload() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            async let topo = SupplierOperationsService.topology()
            async let driverResp = SupplierService.fleetDrivers()
            async let vehicleResp = SupplierService.fleetVehicles()
            async let orgResp = SupplierOperationsService.orgMembers()
            topology = try await topo
            drivers = try await driverResp
            vehicles = try await vehicleResp
            orgMembers = try await orgResp
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func deactivateMember(_ userId: String) async {
        memberActionId = userId
        defer { memberActionId = nil }
        do {
            orgMembers = try await SupplierOperationsService.deactivateOrgMember(
                userId,
                idempotencyKey: SupplierIdempotencyKeys.orgMemberDeactivate(scopeId: SupplierIdempotencyKeys.supplierScopeId(), userId: userId)
            )
        } catch {
            self.error = error.localizedDescription
        }
    }
}
