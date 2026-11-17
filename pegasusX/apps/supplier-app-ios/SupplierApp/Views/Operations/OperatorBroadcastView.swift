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
                SupplierSectionHeader(title: "mobile_supplier.chrome.operator_broadcast")
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
                Picker("mobile_supplier.chrome.target_role", selection: $broadcastRole) {
                    ForEach(broadcastRoles, id: \.self) { role in
                        Text(role).tag(role)
                    }
                }
                Button(LocalizedStringKey(broadcasting ? "mobile_supplier.chrome.sending" : "mobile_supplier.chrome.send_broadcast")) {
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
            bodyText = template.body.replacingOccurrences(of: "{date}", with: date.isEmpty ? L10n.format("mobile_supplier.chrome.the_selected_date") : date)
        } else {
            bodyText = template.body
        }
    }
}
