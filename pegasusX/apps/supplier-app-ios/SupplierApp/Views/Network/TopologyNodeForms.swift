import SwiftUI

enum TopologyMutation {
  static let defaultLat = 41.2995
  static let defaultLng = 69.2401

  @MainActor
  static func appendWarehouse(
    name: String,
    lat: Double,
    lng: Double,
    coverageRadiusKm: Double = 50
  ) async throws -> SupplierTopologyResponse {
    let topology = try await SupplierOperationsService.topology()
    var warehouses = topology.warehouses.map { node in
      SupplierTopologyWarehouseInput(
        warehouseId: node.warehouseId.isEmpty ? nil : node.warehouseId,
        name: node.name,
        lat: node.lat,
        lng: node.lng,
        coverageRadiusKm: node.coverageRadiusKm,
        isActive: node.isActive,
        isOnShift: node.isOnShift,
        transferMode: (node.transferMode ?? "").isEmpty ? "TRUCK" : (node.transferMode ?? "TRUCK")
      )
    }
    warehouses.append(
      SupplierTopologyWarehouseInput(
        warehouseId: nil,
        name: name.trimmingCharacters(in: .whitespacesAndNewlines),
        lat: lat,
        lng: lng,
        coverageRadiusKm: coverageRadiusKm,
        isActive: true,
        isOnShift: true,
        transferMode: "TRUCK"
      )
    )
    let factories = topology.factories.map { node in
      SupplierTopologyFactoryInput(
        factoryId: node.factoryId.isEmpty ? nil : node.factoryId,
        name: node.name,
        lat: node.lat,
        lng: node.lng,
        isActive: node.isActive
      )
    }
    return try await SupplierOperationsService.updateTopology(
      SupplierTopologyUpdateRequest(warehouses: warehouses, factories: factories)
    )
  }

  @MainActor
  static func appendFactory(name: String, lat: Double, lng: Double) async throws -> SupplierTopologyResponse {
    let topology = try await SupplierOperationsService.topology()
    let warehouses = topology.warehouses.map { node in
      SupplierTopologyWarehouseInput(
        warehouseId: node.warehouseId.isEmpty ? nil : node.warehouseId,
        name: node.name,
        lat: node.lat,
        lng: node.lng,
        coverageRadiusKm: node.coverageRadiusKm,
        isActive: node.isActive,
        isOnShift: node.isOnShift,
        transferMode: (node.transferMode ?? "").isEmpty ? "TRUCK" : (node.transferMode ?? "TRUCK")
      )
    }
    var factories = topology.factories.map { node in
      SupplierTopologyFactoryInput(
        factoryId: node.factoryId.isEmpty ? nil : node.factoryId,
        name: node.name,
        lat: node.lat,
        lng: node.lng,
        isActive: node.isActive
      )
    }
    factories.append(
      SupplierTopologyFactoryInput(
        factoryId: nil,
        name: name.trimmingCharacters(in: .whitespacesAndNewlines),
        lat: lat,
        lng: lng,
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
  @State private var lat = String(TopologyMutation.defaultLat)
  @State private var lng = String(TopologyMutation.defaultLng)
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
          TextField("Name", text: $name)
          TextField("Latitude", text: $lat).keyboardType(.decimalPad)
          TextField("Longitude", text: $lng).keyboardType(.decimalPad)
          TextField("Coverage radius (km)", text: $coverageKm).keyboardType(.decimalPad)
        }
        Section {
          Text("Set coordinates for your distribution node. You can refine location later in topology.")
            .font(.caption)
            .foregroundStyle(.secondary)
        }
      }
      .navigationTitle("Add warehouse")
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("Cancel") { dismiss() }
        }
        ToolbarItem(placement: .confirmationAction) {
          Button(saving ? "Saving…" : "Save") { Task { await save() } }
            .disabled(saving || name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
      }
    }
  }

  @MainActor
  private func save() async {
    guard let latValue = Double(lat), let lngValue = Double(lng) else {
      error = "Latitude and longitude must be numbers."
      return
    }
    saving = true
    error = nil
    defer { saving = false }
    do {
      _ = try await TopologyMutation.appendWarehouse(
        name: name,
        lat: latValue,
        lng: lngValue,
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
  @State private var lat = "41.3111"
  @State private var lng = "69.2797"
  @State private var saving = false
  @State private var error: String?

  var body: some View {
    NavigationStack {
      Form {
        if let error {
          Section { Text(error).foregroundStyle(.red) }
        }
        Section("Factory") {
          TextField("Name", text: $name)
          TextField("Latitude", text: $lat).keyboardType(.decimalPad)
          TextField("Longitude", text: $lng).keyboardType(.decimalPad)
        }
      }
      .navigationTitle("Add factory")
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("Cancel") { dismiss() }
        }
        ToolbarItem(placement: .confirmationAction) {
          Button(saving ? "Saving…" : "Save") { Task { await save() } }
            .disabled(saving || name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
        }
      }
    }
  }

  @MainActor
  private func save() async {
    guard let latValue = Double(lat), let lngValue = Double(lng) else {
      error = "Latitude and longitude must be numbers."
      return
    }
    saving = true
    error = nil
    defer { saving = false }
    do {
      _ = try await TopologyMutation.appendFactory(name: name, lat: latValue, lng: lngValue)
      onSaved()
      dismiss()
    } catch {
      self.error = error.localizedDescription
    }
  }
}
