import Observation
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
        userInfo: [NSLocalizedDescriptionKey: "mobile_supplier.chrome.add_warehouse_before_factories"]
      )
    }
    return try await SupplierOperationsService.updateTopology(
      SupplierTopologyUpdateRequest(warehouses: warehouses, factories: factories)
    )
  }
}

@Observable
@MainActor
final class TopologyNodeFormViewModel {
  var name = ""
  var location = AddressLocationValue(lat: TopologyMutation.defaultLat, lng: TopologyMutation.defaultLng)
  var coverageKm = "50"
  var saving = false
  var error: String?

  var canSave: Bool {
    !saving && !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !location.address.isEmpty
  }

  func saveWarehouse() async -> Bool {
    saving = true
    error = nil
    defer { saving = false }
    do {
      _ = try await TopologyMutation.appendWarehouse(
        name: name,
        location: location,
        coverageRadiusKm: Double(coverageKm) ?? 50
      )
      PegasusHaptics.success()
      return true
    } catch {
      PegasusHaptics.error()
      SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
      return false
    }
  }

  func saveFactory() async -> Bool {
    saving = true
    error = nil
    defer { saving = false }
    do {
      _ = try await TopologyMutation.appendFactory(name: name, location: location)
      PegasusHaptics.success()
      return true
    } catch {
      PegasusHaptics.error()
      SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
      return false
    }
  }
}

struct AddWarehouseSheet: View {
  @Environment(\.dismiss) private var dismiss
  var onSaved: () -> Void
  @State private var vm = TopologyNodeFormViewModel()

  var body: some View {
    @Bindable var vm = vm
    NavigationStack {
      Form {
        if let error = vm.error {
          Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
        }
        Section("mobile_supplier.chrome.warehouse") {
          TextField("retailer_desktop.pos.text.name", text: $vm.name)
          AddressLocationField(value: $vm.location, label: "mobile_supplier.chrome.warehouse_address")
          TextField("supplier_portal.residual.text.coverage_radius_km", text: $vm.coverageKm).keyboardType(.decimalPad)
        }
      }
      .scrollDismissesKeyboard(.interactively)
      .navigationTitle("supplier_portal.warehouses.components.warehouse_form.text.add_warehouse")
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("common.action.cancel") { dismiss() }
        }
        ToolbarItem(placement: .confirmationAction) {
          Button(LocalizedStringKey(vm.saving ? "mobile_supplier.chrome.saving" : "mobile_supplier.chrome.save")) {
            Task {
              if await vm.saveWarehouse() {
                onSaved()
                dismiss()
              }
            }
          }
          .disabled(!vm.canSave)
        }
      }
    }
  }
}

struct AddFactorySheet: View {
  @Environment(\.dismiss) private var dismiss
  var onSaved: () -> Void
  @State private var vm = TopologyNodeFormViewModel()

  var body: some View {
    @Bindable var vm = vm
    NavigationStack {
      Form {
        if let error = vm.error {
          Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
        }
        Section("mobile_supplier.chrome.factory") {
          TextField("retailer_desktop.pos.text.name", text: $vm.name)
          AddressLocationField(value: $vm.location, label: "mobile_supplier.chrome.factory_address")
        }
      }
      .scrollDismissesKeyboard(.interactively)
      .navigationTitle("supplier_portal.factories.components.factory_form.text.add_factory")
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("common.action.cancel") { dismiss() }
        }
        ToolbarItem(placement: .confirmationAction) {
          Button(LocalizedStringKey(vm.saving ? "mobile_supplier.chrome.saving" : "mobile_supplier.chrome.save")) {
            Task {
              if await vm.saveFactory() {
                onSaved()
                dismiss()
              }
            }
          }
          .disabled(!vm.canSave)
        }
      }
    }
  }
}
