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
                        Text("Drivers (\(drivers.count))").tag(0)
                        Text("Vehicles (\(vehicles.count))").tag(1)
                        Text("Org (\(orgMembers.count))").tag(2)
                    }
                    .pickerStyle(.segmented)
                    .padding()

                    switch tab {
                    case 0: driverList
                    case 1: vehicleList
                    default: orgList
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
    }

    private var driverList: some View {
        Group {
            if drivers.isEmpty {
                SupplierEmptyView(title: "No drivers", message: "Create a driver to start fleet onboarding.")
            } else {
                List(drivers) { driver in
                    VStack(alignment: .leading) {
                        Text(driver.name).font(.headline)
                        Text("\(nodeLabel(type: driver.homeNodeType, id: driver.homeNodeId)) · \(driver.phone)")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    private var vehicleList: some View {
        Group {
            if vehicles.isEmpty {
                SupplierEmptyView(title: "No vehicles", message: "Create a vehicle for driver assignment.")
            } else {
                List(vehicles) { vehicle in
                    VStack(alignment: .leading) {
                        Text(vehicle.label ?? vehicle.licensePlate).font(.headline)
                        Text("\(vehicle.licensePlate) · \(nodeLabel(type: vehicle.homeNodeType, id: vehicle.homeNodeId))")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    private var orgList: some View {
        Group {
            if orgMembers.isEmpty {
                SupplierEmptyView(title: "No org members", message: "Create warehouse, factory, or payload staff.")
            } else {
                List(orgMembers) { member in
                    VStack(alignment: .leading) {
                        Text(member.name).font(.headline)
                        Text("\(member.supplierRole) · \(member.phone) · \(member.isActive ? "Active" : "Inactive")")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                        if member.isActive {
                            Button(role: .destructive) {
                                Task { await deactivateMember(member.userId) }
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

    private func nodeLabel(type: String, id: String) -> String {
        guard let topology else { return id }
        if type == "FACTORY" {
            return topology.factories.first { $0.factoryId == id }?.name ?? id
        }
        return topology.warehouses.first { $0.warehouseId == id }?.name ?? id
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
            orgMembers = try await SupplierOperationsService.deactivateOrgMember(userId, idempotencyKey: UUID().uuidString)
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct CreateDriverSheet: View {
    let topology: SupplierTopologyResponse
    let vehicles: [FleetVehicle]
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var phone = ""
    @State private var pin = ""
    @State private var nodeType = "WAREHOUSE"
    @State private var nodeId = ""
    @State private var vehicleId = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("Name", text: $name)
                TextField("Phone", text: $phone)
                SecureField("PIN", text: $pin)
                Picker("Node type", selection: $nodeType) {
                    Text("Warehouse").tag("WAREHOUSE")
                    Text("Factory").tag("FACTORY")
                }
                .onChange(of: nodeType) { _, _ in nodeId = ""; vehicleId = "" }
                Picker("Home node", selection: $nodeId) {
                    Text("Select node").tag("")
                    ForEach(nodeOptions, id: \.0) { id, label in
                        Text(label).tag(id)
                    }
                }
                if !filteredVehicles.isEmpty {
                    Picker("Vehicle", selection: $vehicleId) {
                        Text("Assign later").tag("")
                        ForEach(filteredVehicles) { vehicle in
                            Text(vehicle.licensePlate).tag(vehicle.vehicleId)
                        }
                    }
                }
                if let error { Text(error).foregroundStyle(.red) }
            }
            .navigationTitle("Create driver")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? "…" : "Create") { Task { await create() } }
                        .disabled(busy || name.isEmpty || phone.isEmpty || pin.isEmpty || nodeId.isEmpty)
                }
            }
        }
    }

    private var nodeOptions: [(String, String)] {
        if nodeType == "FACTORY" {
            return topology.factories.map { ($0.factoryId, $0.name) }
        }
        return topology.warehouses.map { ($0.warehouseId, $0.name) }
    }

    private var filteredVehicles: [FleetVehicle] {
        vehicles.filter { $0.homeNodeType == nodeType && $0.homeNodeId == nodeId }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let request = FleetDriverCreateRequest(
                name: name.trimmingCharacters(in: .whitespaces),
                phone: phone.trimmingCharacters(in: .whitespaces),
                pin: pin,
                homeNodeType: nodeType,
                homeNodeId: nodeId,
                vehicleId: vehicleId.isEmpty ? nil : vehicleId,
                isActive: nil
            )
            _ = try await SupplierOperationsService.createFleetDriver(request, idempotencyKey: UUID().uuidString)
            dismiss()
            onDone()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct CreateVehicleSheet: View {
    let topology: SupplierTopologyResponse
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var label = ""
    @State private var plate = ""
    @State private var nodeType = "WAREHOUSE"
    @State private var nodeId = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("Label", text: $label)
                TextField("License plate", text: $plate)
                    .textInputAutocapitalization(.characters)
                Picker("Node type", selection: $nodeType) {
                    Text("Warehouse").tag("WAREHOUSE")
                    Text("Factory").tag("FACTORY")
                }
                .onChange(of: nodeType) { _, _ in nodeId = "" }
                Picker("Home node", selection: $nodeId) {
                    Text("Select node").tag("")
                    ForEach(nodeOptions, id: \.0) { id, name in Text(name).tag(id) }
                }
                if let error { Text(error).foregroundStyle(.red) }
            }
            .navigationTitle("Create vehicle")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? "…" : "Create") { Task { await create() } }
                        .disabled(busy || plate.isEmpty || nodeId.isEmpty)
                }
            }
        }
    }

    private var nodeOptions: [(String, String)] {
        if nodeType == "FACTORY" {
            return topology.factories.map { ($0.factoryId, $0.name) }
        }
        return topology.warehouses.map { ($0.warehouseId, $0.name) }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        do {
            let request = FleetVehicleCreateRequest(
                label: label.isEmpty ? nil : label,
                licensePlate: plate.uppercased(),
                homeNodeType: nodeType,
                homeNodeId: nodeId,
                isActive: nil
            )
            _ = try await SupplierOperationsService.createFleetVehicle(request, idempotencyKey: UUID().uuidString)
            dismiss()
            onDone()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct CreateOrgMemberSheet: View {
    let topology: SupplierTopologyResponse
    let onDone: () -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var email = ""
    @State private var phone = ""
    @State private var password = ""
    @State private var role = "WAREHOUSE_ADMIN"
    @State private var nodeType = "WAREHOUSE"
    @State private var nodeId = ""
    @State private var busy = false
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Form {
                TextField("Name", text: $name)
                TextField("Email", text: $email)
                TextField("Phone", text: $phone)
                SecureField("Password", text: $password)
                Picker("Role", selection: $role) {
                    Text("Warehouse admin").tag("WAREHOUSE_ADMIN")
                    Text("Factory admin").tag("FACTORY_ADMIN")
                    Text("Payload staff").tag("PAYLOAD")
                    Text("Supplier operator").tag("ADMIN")
                }
                .onChange(of: role) { _, _ in nodeId = "" }
                if role == "PAYLOAD" {
                    Picker("Node type", selection: $nodeType) {
                        Text("Warehouse").tag("WAREHOUSE")
                        Text("Factory").tag("FACTORY")
                    }
                    .onChange(of: nodeType) { _, _ in nodeId = "" }
                }
                if role != "ADMIN" {
                    Picker("Node", selection: $nodeId) {
                        Text("Select node").tag("")
                        ForEach(nodeOptions, id: \.0) { id, label in Text(label).tag(id) }
                    }
                }
                if let error { Text(error).foregroundStyle(.red) }
            }
            .navigationTitle("Create org member")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button(busy ? "…" : "Create") { Task { await create() } }
                        .disabled(busy || name.isEmpty || phone.isEmpty || password.isEmpty || (role != "ADMIN" && nodeId.isEmpty))
                }
            }
        }
    }

    private var nodeOptions: [(String, String)] {
        switch role {
        case "FACTORY_ADMIN":
            return topology.factories.map { ($0.factoryId, $0.name) }
        case "ADMIN":
            return []
        default:
            if nodeType == "FACTORY" {
                return topology.factories.map { ($0.factoryId, $0.name) }
            }
            return topology.warehouses.map { ($0.warehouseId, $0.name) }
        }
    }

    private func create() async {
        busy = true
        error = nil
        defer { busy = false }
        let warehouseId: String? = {
            if role == "WAREHOUSE_ADMIN" { return nodeId }
            if role == "PAYLOAD" && nodeType == "WAREHOUSE" { return nodeId }
            return nil
        }()
        let factoryId: String? = {
            if role == "FACTORY_ADMIN" { return nodeId }
            if role == "PAYLOAD" && nodeType == "FACTORY" { return nodeId }
            return nil
        }()
        do {
            let request = SupplierOrgMemberCreateRequest(
                name: name.trimmingCharacters(in: .whitespaces),
                email: email.isEmpty ? nil : email,
                phone: phone.trimmingCharacters(in: .whitespaces),
                password: password,
                supplierRole: role,
                assignedWarehouseId: warehouseId,
                assignedFactoryId: factoryId
            )
            _ = try await SupplierOperationsService.createOrgMember(request, idempotencyKey: UUID().uuidString)
            dismiss()
            onDone()
        } catch {
            self.error = error.localizedDescription
        }
    }
}
