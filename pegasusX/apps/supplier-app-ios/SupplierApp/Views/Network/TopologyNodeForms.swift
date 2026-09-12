import SwiftUI

enum TopologyMutation {
  static var defaultLat: Double { packMapCoordinate().lat }
  static var defaultLng: Double { packMapCoordinate().lng }

  static func warehouseInput(
    from node: SupplierTopologyWarehouse? = nil,
    warehouseId: String?,
    name: String,
    location: AddressLocationValue,
    coverageRadiusKm: Double,
    isActive: Bool,
    isOnShift: Bool,
    transferMode: String
  ) -> SupplierTopologyWarehouseInput {
    var input = SupplierTopologyWarehouseInput(
      warehouseId: warehouseId?.isEmpty == true ? nil : warehouseId,
      name: name,
      address: location.address.isEmpty ? nil : location.address,
      placeId: location.placeId,
      lat: location.lat,
      lng: location.lng,
      coverageRadiusKm: coverageRadiusKm,
      isActive: isActive,
      isOnShift: isOnShift,
      transferMode: transferMode
    )
    if let node {
      input.primaryFactoryId = node.primaryFactoryId.flatMap { $0.isEmpty ? nil : $0 }
      input.countryCode = node.countryCode.isEmpty ? nil : node.countryCode
      input.coverageCities = node.coverageCities.isEmpty ? nil : node.coverageCities
      input.assignedFactoryIds = node.assignedFactoryIds.isEmpty ? nil : node.assignedFactoryIds
    }
    return input
  }

  static func warehouseInput(from node: SupplierTopologyWarehouse) -> SupplierTopologyWarehouseInput {
    warehouseInput(
      from: node,
      warehouseId: node.warehouseId.isEmpty ? nil : node.warehouseId,
      name: node.name,
      location: AddressLocationValue(address: node.address, lat: node.lat, lng: node.lng, placeId: node.placeId),
      coverageRadiusKm: node.coverageRadiusKm,
      isActive: node.isActive,
      isOnShift: node.isOnShift,
      transferMode: (node.transferMode ?? "").isEmpty ? "TRUCK" : (node.transferMode ?? "TRUCK")
    )
  }

  static func factoryInput(
    from node: SupplierTopologyFactory? = nil,
    factoryId: String?,
    name: String,
    location: AddressLocationValue,
    isActive: Bool
  ) -> SupplierTopologyFactoryInput {
    var input = SupplierTopologyFactoryInput(
      factoryId: factoryId?.isEmpty == true ? nil : factoryId,
      name: name,
      address: location.address.isEmpty ? nil : location.address,
      placeId: location.placeId,
      lat: location.lat,
      lng: location.lng,
      isActive: isActive
    )
    if let node, !node.countryCode.isEmpty {
      input.countryCode = node.countryCode
    }
    return input
  }

  static func factoryInput(from node: SupplierTopologyFactory) -> SupplierTopologyFactoryInput {
    factoryInput(
      from: node,
      factoryId: node.factoryId.isEmpty ? nil : node.factoryId,
      name: node.name,
      location: AddressLocationValue(address: node.address, lat: node.lat, lng: node.lng, placeId: node.placeId),
      isActive: node.isActive
    )
  }

  @MainActor
  static func appendWarehouse(
    name: String,
    location: AddressLocationValue,
    coverageRadiusKm: Double = 50
  ) async throws -> SupplierTopologyResponse {
    let topology = try await SupplierOperationsService.topology()
    var warehouses = topology.warehouses.map { warehouseInput(from: $0) }
    warehouses.append(
      warehouseInput(
        warehouseId: nil,
        name: name.trimmingCharacters(in: .whitespacesAndNewlines),
        location: location,
        coverageRadiusKm: coverageRadiusKm,
        isActive: true,
        isOnShift: true,
        transferMode: "TRUCK"
      )
    )
    let factories = topology.factories.map { factoryInput(from: $0) }
    return try await SupplierOperationsService.updateTopology(
      SupplierTopologyUpdateRequest(warehouses: warehouses, factories: factories)
    )
  }

  @MainActor
  static func appendFactory(name: String, location: AddressLocationValue) async throws -> SupplierTopologyResponse {
    let topology = try await SupplierOperationsService.topology()
    let warehouses = topology.warehouses.map { warehouseInput(from: $0) }
    var factories = topology.factories.map { factoryInput(from: $0) }
    factories.append(
      factoryInput(
        factoryId: nil,
        name: name.trimmingCharacters(in: .whitespacesAndNewlines),
        location: location,
        isActive: true
      )
    )
    guard !warehouses.isEmpty else {
      throw NSError(
        domain: "TopologyMutation",
        code: 1,
        userInfo: [NSLocalizedDescriptionKey: "Add at least one warehouse before creating factories."]
      )
    }
    return try await SupplierOperationsService.updateTopology(
      SupplierTopologyUpdateRequest(warehouses: warehouses, factories: factories)
    )
  }
}

struct TopologyCenteredEmptyState: View {
  let title: String
  let message: String
  let actionLabel: String
  let action: () -> Void

  var body: some View {
    VStack(spacing: SupplierTheme.spacingLG) {
      Spacer()
      SupplierEmptyView(title: title, message: message)
      Button(actionLabel, action: action)
        .buttonStyle(.borderedProminent)
        .tint(.primary)
      Spacer()
    }
    .frame(maxWidth: .infinity, maxHeight: .infinity)
    .padding()
  }
}

struct AddWarehouseSheet: View {
  @Environment(\.dismiss) private var dismiss
  var onSaved: () -> Void

  @State private var name = ""
  @State private var location = AddressLocationValue(lat: TopologyMutation.defaultLat, lng: TopologyMutation.defaultLng)
  @State private var coverageKm = "50"
  @State private var saving = false
  @State private var error: String?

  var body: some View {
    NavigationStack {
      Form {
        if let error {
          Section { Text(error).foregroundStyle(.red) }
        }
        Section("Warehouse") {
          TextField("retailer_desktop.pos.text.name", text: $name)
          AddressLocationField(value: $location, label: "Warehouse address")
          TextField("supplier_portal.residual.text.coverage_radius_km", text: $coverageKm).keyboardType(.decimalPad)
        }
      }
      .navigationTitle("supplier_portal.warehouses.components.warehouse_form.text.add_warehouse")
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("common.action.cancel") { dismiss() }
        }
        ToolbarItem(placement: .confirmationAction) {
          Button(saving ? "Saving…" : "Save") { Task { await save() } }
            .disabled(saving || name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || location.address.isEmpty)
        }
      }
    }
  }

  @MainActor
  private func save() async {
    saving = true
    error = nil
    defer { saving = false }
    do {
      _ = try await TopologyMutation.appendWarehouse(
        name: name,
        location: location,
        coverageRadiusKm: Double(coverageKm) ?? 50
      )
      onSaved()
      dismiss()
    } catch {
      self.error = error.localizedDescription
    }
  }
}

struct AddFactorySheet: View {
  @Environment(\.dismiss) private var dismiss
  var onSaved: () -> Void

  @State private var name = ""
  @State private var location = AddressLocationValue(lat: TopologyMutation.defaultLat, lng: TopologyMutation.defaultLng)
  @State private var saving = false
  @State private var error: String?

  var body: some View {
    NavigationStack {
      Form {
        if let error {
          Section { Text(error).foregroundStyle(.red) }
        }
        Section("Factory") {
          TextField("retailer_desktop.pos.text.name", text: $name)
          AddressLocationField(value: $location, label: "Factory address")
        }
      }
      .navigationTitle("supplier_portal.factories.components.factory_form.text.add_factory")
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("common.action.cancel") { dismiss() }
        }
        ToolbarItem(placement: .confirmationAction) {
          Button(saving ? "Saving…" : "Save") { Task { await save() } }
            .disabled(saving || name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || location.address.isEmpty)
        }
      }
    }
  }

  @MainActor
  private func save() async {
    saving = true
    error = nil
    defer { saving = false }
    do {
      _ = try await TopologyMutation.appendFactory(name: name, location: location)
      onSaved()
      dismiss()
    } catch {
      self.error = error.localizedDescription
    }
  }
}
