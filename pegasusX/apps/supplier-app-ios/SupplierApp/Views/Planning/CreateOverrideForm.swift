import SwiftUI

struct CreateOverrideForm: View {
    let templates: [SeasonalBuiltinTemplate]
    @Binding var templateId: String
    @Binding var name: String
    @Binding var startDate: String
    @Binding var endDate: String
    let formError: String?
    let saving: Bool
    let onSubmit: () -> Void

    var body: some View {
        if !templates.isEmpty {
            Picker("Template", selection: $templateId) {
                Text("Custom").tag("")
                ForEach(templates) { template in
                    Text(template.name).tag(template.id)
                }
            }
        }
        TextField("Name (optional)", text: $name)
        TextField("Start (YYYY-MM-DD)", text: $startDate)
            .textInputAutocapitalization(.never)
        TextField("End (YYYY-MM-DD)", text: $endDate)
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
