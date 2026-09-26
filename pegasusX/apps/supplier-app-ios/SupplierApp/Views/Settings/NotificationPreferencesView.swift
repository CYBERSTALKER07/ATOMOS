import SwiftUI

struct NotificationPreferencesView: View {
    @State private var prefs: [NotificationPreferenceRow] = []
    @State private var loading = true
    @State private var saved = false

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading preferences…")
            } else {
                List {
                    if saved {
                        Text("mobile_supplier.ui.saved").foregroundStyle(SupplierTheme.success)
                    }
                    ForEach(prefs.indices, id: \.self) { idx in
                        let p = prefs[idx]
                        VStack(alignment: .leading) {
                            Text(p.eventType).font(.caption.monospaced())
                            Text(p.channel).font(.caption)
                            Toggle("Enabled", isOn: Binding(
                                get: { prefs[idx].enabled },
                                set: { newVal in
                                    prefs[idx] = NotificationPreferenceRow(
                                        eventType: p.eventType,
                                        channel: p.channel,
                                        enabled: newVal
                                    )
                                }
                            ))
                        }
                    }
                }
            }
        }
        .navigationTitle("supplier_portal.settings.notification_preferences.text.notification_preferences")
        .toolbar {
            Button("common.action.save") {
                Task {
                    _ = try? await SupplierOperationsService.patchNotificationPreferences(prefs)
                    saved = true
                }
            }
        }
        .task {
            loading = true
            if let resp = try? await SupplierOperationsService.notificationPreferences() {
                prefs = resp.preferences
            }
            loading = false
        }
    }
}
