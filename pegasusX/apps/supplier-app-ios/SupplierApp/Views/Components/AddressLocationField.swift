import SwiftUI

struct AddressLocationValue: Equatable {
  var address: String = ""
  var lat: Double = 0
  var lng: Double = 0
  var placeId: String?
}

struct AddressLocationField: View {
  @Binding var value: AddressLocationValue
  var label: String = "mobile_supplier.chrome.address"

  @State private var vm = AddressLocationViewModel()

  var body: some View {
    @Bindable var vm = vm
    VStack(alignment: .leading, spacing: 8) {
      TextField(LocalizedStringKey(label), text: $vm.query)
        .textInputAutocapitalization(.words)
        .onChange(of: vm.query) { _, text in
          vm.scheduleSearch(text)
        }
        .onDisappear {
          vm.cancelSearch()
        }
      if !vm.suggestions.isEmpty {
        ForEach(vm.suggestions.prefix(5), id: \.placeId) { item in
          Button(item.description) {
            PegasusHaptics.selection()
            Task {
              if let next = await vm.pick(placeId: item.placeId, fallback: item.description) {
                value = next
              }
            }
          }
          .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
        }
      }
      Button(LocalizedStringKey(vm.locating ? "mobile_supplier.chrome.locating" : "mobile_supplier.chrome.share_my_location")) {
        PegasusHaptics.light()
        Task {
          if let next = await vm.useMyLocation() {
            value = next
          }
        }
      }
      .frame(minHeight: 44)
      .disabled(vm.locating)
      if let error = vm.error {
        Text(error).foregroundStyle(SupplierTheme.destructive).font(.caption)
      }
    }
    .onAppear { vm.query = value.address }
    .onChange(of: value.address) { _, next in
      if vm.query != next { vm.query = next }
    }
  }
}
