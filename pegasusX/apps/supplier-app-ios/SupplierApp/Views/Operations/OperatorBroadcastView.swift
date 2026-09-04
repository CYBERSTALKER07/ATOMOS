import SwiftUI

struct OperatorBroadcastView: View {
    let broadcastRoles = ["ALL", "DRIVER", "RETAILER", "PAYLOAD"]

    @Binding var title: String
    @Binding var bodyText: String
    @Binding var broadcastRole: String
    @Binding var templateDate: String
    let broadcasting: Bool
    let sendBroadcast: () -> Void

    var body: some View {
        Group {
            Section {
                SupplierSectionHeader(title: "Operator broadcast")
            }
            Section {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: SupplierTheme.spacingSM) {
                        ForEach(supplierBroadcastTemplates) { template in
                            Button(template.title) {
                                applyTemplate(template)
                            }
                            .buttonStyle(.bordered)
                            .font(.caption)
                        }
                    }
                }
                TextField("mobile_supplier.ui.closure_effective_date_optional", text: $templateDate)
                TextField("supplier_portal.admin.control_center.field.title", text: $title)
                TextField("warehouse_portal.residual.text.message", text: $bodyText, axis: .vertical)
                    .lineLimit(3...6)
                Picker("Target role", selection: $broadcastRole) {
                    ForEach(broadcastRoles, id: \.self) { role in
                        Text(role).tag(role)
                    }
                }
                Button(broadcasting ? "Sending…" : "Send broadcast") {
                    sendBroadcast()
                }
                .disabled(broadcasting || title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                    || bodyText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
    }

    private func applyTemplate(_ template: SupplierBroadcastTemplate) {
        title = template.title
        broadcastRole = template.defaultRole
        let date = templateDate.trimmingCharacters(in: .whitespacesAndNewlines)
        if template.body.contains("{date}") {
            bodyText = template.body.replacingOccurrences(of: "{date}", with: date.isEmpty ? "the selected date" : date)
        } else {
            bodyText = template.body
        }
    }
}
