import SwiftUI

struct CreateOverrideForm: View {
    let templates: [SeasonalBuiltinTemplate]
    @Binding var templateId: String
    @Binding var name: String
    @Binding var startDate: String
    @Binding var endDate: String
    @Binding var multiplier: String
    let formError: String?
    let saving: Bool
    let onSubmit: () -> Void

    var body: some View {
        if !templates.isEmpty {
            Picker("Template", selection: $templateId) {
                Text("supplier_portal.settings.planning.create_override_form.text.custom").tag("")
                ForEach(templates) { template in
                    if let m = template.multiplier {
                        Text("\(template.name) (×\(m))").tag(template.id)
                    } else {
                        Text(template.name).tag(template.id)
                    }
                }
            }
            .onChange(of: templateId) { _, newValue in
                if let tpl = templates.first(where: { $0.id == newValue }), let m = tpl.multiplier {
                    multiplier = String(m)
                } else if newValue.isEmpty {
                    multiplier = ""
                }
            }
        }
        TextField("mobile_supplier.ui.name_optional", text: $name)
        TextField("mobile_supplier.ui.start_yyyy_mm_dd", text: $startDate)
            .textInputAutocapitalization(.never)
        TextField("mobile_supplier.ui.end_yyyy_mm_dd", text: $endDate)
            .textInputAutocapitalization(.never)
        TextField("Multiplier (optional)", text: $multiplier)
            .keyboardType(.decimalPad)
            .textInputAutocapitalization(.never)
        if let formError {
            Text(formError)
                .font(.caption)
                .foregroundStyle(SupplierTheme.destructive)
        }
        Button(saving ? "Saving…" : "Create override") {
            onSubmit()
        }
        .disabled(saving)
    }
}
