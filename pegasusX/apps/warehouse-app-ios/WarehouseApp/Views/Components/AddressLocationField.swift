import CoreLocation
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

    @State private var vm = AddressLocationFieldViewModel()
    @FocusState private var focused: Bool

    private var pinned: Bool {
        GeocodeLocationSupport.hasValidCoordinates(lat: value.lat, lng: value.lng)
    }

    var body: some View {
        @Bindable var vm = vm
        VStack(alignment: .leading, spacing: 8) {
            TextField(label, text: $vm.query)
                .textInputAutocapitalization(.words)
                .focused($focused)
                .onChange(of: vm.query) { _, text in
                    value.address = text
                    vm.scheduleSearch(text)
                }
                .onDisappear {
                    vm.cancelSearch()
                }
                .onChange(of: focused) { _, isFocused in
                    if !isFocused {
                        Task {
                            value = await vm.resolveOnBlur(currentValue: value)
                        }
                    }
                }
            ForEach(vm.suggestions.prefix(5), id: \.description) { item in
                Button(item.description) {
                    PegasusHaptics.selection()
                    Task {
                        value = await vm.pick(placeId: item.placeId, fallback: item.description, currentValue: value)
                    }
                }
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                .contentShape(Rectangle())
                .accessibilityLabel("Suggested address: \(item.description)")
            }
            Button(vm.locating ? "Locating…" : "Share my location") {
                PegasusHaptics.light()
                Task {
                    value = await vm.useMyLocation(currentValue: value)
                }
            }
            .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
            .contentShape(Rectangle())
            .disabled(vm.locating || vm.resolving)
            .accessibilityLabel(vm.locating ? "Locating your warehouse" : "Share my location")
            if pinned {
                Text("supplier_portal.location_picker.text.pinned_for_dispatch_routing")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else if vm.resolving {
                Text("factory_portal.location_picker.text.resolving_address")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if let error = vm.error {
                Text(error).foregroundStyle(.red).font(.caption)
            }
        }
        .onAppear { vm.syncExternalAddress(value.address) }
        .onChange(of: value.address) { _, next in
            vm.syncExternalAddress(next)
        }
    }
}
