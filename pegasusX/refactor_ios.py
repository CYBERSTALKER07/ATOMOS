import os
import re

base_path = '/Users/shakhzod/ATOMOS/pegasusX/apps/supplier-app-ios/SupplierApp/Views/OrgFleet'
components_path = os.path.join(base_path, 'Components')

with open(os.path.join(base_path, 'OrgFleetView.swift'), 'r') as f:
    content = f.read()

# Utils
utils_content = """import Foundation

func nodeLabel(topology: SupplierTopologyResponse?, type: String, id: String) -> String {
    guard let topology else { return id }
    if type == "FACTORY" {
        return topology.factories.first { $0.factoryId == id }?.name ?? id
    }
    return topology.warehouses.first { $0.warehouseId == id }?.name ?? id
}
"""
with open(os.path.join(components_path, 'OrgFleetUtils.swift'), 'w') as f:
    f.write(utils_content)

# Extract sheets
def extract_struct(name):
    pattern = r"private struct " + name + r".*?^\}"
    match = re.search(pattern, content, re.MULTILINE | re.DOTALL)
    if match:
        struct_content = match.group(0).replace(f"private struct {name}", f"struct {name}")
        with open(os.path.join(components_path, f"{name}.swift"), 'w') as f:
            f.write("import SwiftUI\n\n" + struct_content + "\n")
    else:
        print(f"Could not find {name}")

extract_struct("CreateDriverSheet")
extract_struct("CreateVehicleSheet")
extract_struct("CreateOrgMemberSheet")
extract_struct("EditOrgMemberSheet")

# Lists
driver_list = """import SwiftUI

struct DriverListView: View {
    let drivers: [FleetDriver]
    let topology: SupplierTopologyResponse?
    
    var body: some View {
        Group {
            if drivers.isEmpty {
                SupplierEmptyView(title: "No drivers", message: "Create a driver to start fleet onboarding.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(drivers) { driver in
                        VStack(alignment: .leading) {
                            Text(driver.name).font(.headline)
                            Text("\\(nodeLabel(topology: topology, type: driver.homeNodeType, id: driver.homeNodeId)) · \\(driver.phone)")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
    }
}
"""
with open(os.path.join(components_path, 'DriverListView.swift'), 'w') as f:
    f.write(driver_list)

vehicle_list = """import SwiftUI

struct VehicleListView: View {
    let vehicles: [FleetVehicle]
    let topology: SupplierTopologyResponse?
    
    var body: some View {
        Group {
            if vehicles.isEmpty {
                SupplierEmptyView(title: "No vehicles", message: "Create a vehicle for driver assignment.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vehicles) { vehicle in
                        VStack(alignment: .leading) {
                            Text(vehicle.label ?? vehicle.licensePlate).font(.headline)
                            Text("\\(vehicle.licensePlate) · \\(nodeLabel(topology: topology, type: vehicle.homeNodeType, id: vehicle.homeNodeId))")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            }
        }
    }
}
"""
with open(os.path.join(components_path, 'VehicleListView.swift'), 'w') as f:
    f.write(vehicle_list)

org_member_list = """import SwiftUI

struct OrgMemberListView: View {
    let orgMembers: [SupplierOrgMember]
    let onEdit: (SupplierOrgMember) -> Void
    let onDeactivate: (String) async -> Void
    let memberActionId: String?
    
    var body: some View {
        Group {
            if orgMembers.isEmpty {
                SupplierEmptyView(title: "No org members", message: "Create warehouse, factory, or payload staff.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(orgMembers) { member in
                        VStack(alignment: .leading) {
                            Text(member.name).font(.headline)
                            Text("\\(member.supplierRole) · \\(member.phone) · \\(member.isActive ? "Active" : "Inactive")")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                        .contentShape(Rectangle())
                        .onTapGesture {
                            onEdit(member)
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if member.isActive {
                                Button(role: .destructive) {
                                    Task { await onDeactivate(member.userId) }
                                } label: {
                                    Text("Deactivate")
                                }
                                .disabled(memberActionId == member.userId)
                            }
                        }
                    }
                }
            }
        }
    }
}
"""
with open(os.path.join(components_path, 'OrgMemberListView.swift'), 'w') as f:
    f.write(org_member_list)

main_view = """import SwiftUI

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
                    Picker("Section", selection: $tab) {
                        Text("Drivers (\\(drivers.count))").tag(0)
                        Text("Vehicles (\\(vehicles.count))").tag(1)
                        Text("Org (\\(orgMembers.count))").tag(2)
                    }
                    .pickerStyle(.segmented)
                    .padding()

                    switch tab {
                    case 0: DriverListView(drivers: drivers, topology: topology)
                    case 1: VehicleListView(vehicles: vehicles, topology: topology)
                    default: OrgMemberListView(orgMembers: orgMembers, onEdit: { member in
                        editingMember = member
                        showEditMemberSheet = true
                    }, onDeactivate: { userId in
                        await deactivateMember(userId)
                    }, memberActionId: memberActionId)
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
"""
with open(os.path.join(base_path, 'OrgFleetView.swift'), 'w') as f:
    f.write(main_view)

print("iOS Components created and OrgFleetView refactored!")
