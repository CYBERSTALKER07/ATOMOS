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
                        Text("Saved").foregroundStyle(SupplierTheme.success)
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
        .navigationTitle("Notification preferences")
        .toolbar {
            Button("Save") {
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
