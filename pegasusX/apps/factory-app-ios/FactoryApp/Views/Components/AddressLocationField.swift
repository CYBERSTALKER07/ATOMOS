import SwiftUI

struct AddressLocationValue: Equatable {
    var address: String = ""
    var lat: Double = 0
    var lng: Double = 0
    var placeId: String?
}

struct AddressLocationField: View {
    @Binding var value: AddressLocationValue
    var label: String = "Address"

    @State private var vm = AddressLocationViewModel()
    @FocusState private var focused: Bool

    private var pinned: Bool {
        GeocodeLocationSupport.hasValidCoordinates(lat: value.lat, lng: value.lng)
    }

    var body: some View {
        @Bindable var vm = vm

        VStack(alignment: .leading, spacing: 8) {
            TextField(label, text: $vm.query)
                .textInputAutocapitalization(.words)
                .frame(minHeight: 44)
                .focused($focused)
                .accessibilityLabel(label)
                .onChange(of: vm.query) { _, text in
                    value.address = text
                    vm.scheduleSearch(text)
                }
                .onChange(of: focused) { _, isFocused in
                    if !isFocused {
                        Task {
                            if let resolved = await vm.resolveOnBlur(current: value) {
                                value = resolved
                            }
                        }
                    }
                }

            ForEach(vm.suggestions.prefix(5), id: \.description) { item in
                Button(item.description) {
                    Task {
                        value = await vm.pick(placeId: item.placeId, fallback: item.description, current: value)
                    }
                }
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                .accessibilityLabel("Suggested address: \(item.description)")
            }

            Button(vm.locating ? "Locating…" : "Share my location") {
                Task {
                    if let resolved = await vm.useMyLocation() {
                        value = resolved
                    }
                }
            }
            .frame(minWidth: 44, minHeight: 44, alignment: .leading)
            .disabled(vm.locating || vm.resolving)
            .accessibilityLabel("Share my current GPS location")

            if pinned {
                Text("factory_portal.location_picker.text.pinned_for_supply_routing")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else if vm.resolving {
                Text("factory_portal.location_picker.text.resolving_address")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if let error = vm.error {
                Text(error)
                    .foregroundStyle(.red)
                    .font(.caption)
            }
        }
        .onAppear { vm.syncAddress(value.address) }
        .onDisappear { vm.cancelSearch() }
        .onChange(of: value.address) { _, next in
            vm.syncAddress(next)
        }
    }
}
