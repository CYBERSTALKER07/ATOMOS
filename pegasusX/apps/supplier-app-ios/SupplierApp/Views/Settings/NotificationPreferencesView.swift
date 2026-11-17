import SwiftUI

struct NotificationPreferencesView: View {
    @State private var vm = NotificationPreferencesViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.items.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.preferences")
            } else if let error = vm.error, vm.items.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else {
                List {
                    if vm.saved {
                        Text("mobile_supplier.ui.saved").foregroundStyle(SupplierTheme.success)
                    }
                    if let error = vm.error {
                        Text(error).font(.caption).foregroundStyle(SupplierTheme.destructive)
                    }
                    ForEach($vm.items) { $item in
                        VStack(alignment: .leading) {
                            Text(item.eventType).font(.caption.monospaced())
                            Text(item.channel).font(.caption)
                            Toggle("mobile_supplier.chrome.enabled", isOn: $item.enabled)
                                .onChange(of: item.enabled) { _, _ in
                                    PegasusHaptics.selection()
                                }
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("supplier_portal.settings.notification_preferences.text.notification_preferences")
        .toolbar {
            Button("common.action.save") {
                PegasusHaptics.light()
                Task { await vm.save() }
            }
            .frame(minWidth: 44, minHeight: 44)
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }
}
