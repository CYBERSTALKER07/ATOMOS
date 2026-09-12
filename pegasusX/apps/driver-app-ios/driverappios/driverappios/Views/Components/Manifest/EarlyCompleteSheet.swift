import SwiftUI

struct EarlyCompleteSheet: View {
    let onConfirm: (String, String) -> Void
    @Environment(\.dismiss) private var dismiss
    
    @State private var selectedReason = "FATIGUE"
    @State private var note = ""
    
    private let reasons: [(id: String, label: String)] = [
        ("FATIGUE", "Fatigue / Feeling Unwell"),
        ("TRAFFIC", "Heavy Traffic / Road Block"),
        ("VEHICLE_ISSUE", "Vehicle Issue"),
        ("OTHER", "Other")
    ]
    
    var body: some View {
        NavigationStack {
            List {
                Section {
                    Text("mobile_driver.ui.remaining_orders_will_be_returned_to_the_supplier_for_next_day_r")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                
                Section("Reason") {
                    ForEach(reasons, id: \.id) { reason in
                        Button {
                            selectedReason = reason.id
                        } label: {
                            HStack {
                                Text(reason.label)
                                    .foregroundStyle(LabTheme.fg)
                                Spacer()
                                if selectedReason == reason.id {
                                    Image(systemName: "checkmark")
                                        .foregroundStyle(.blue)
                                        .fontWeight(.semibold)
                                }
                            }
                        }
                    }
                }
                
                Section("Note (optional)") {
                    TextField("mobile_driver.ui.add_a_note", text: $note, axis: .vertical)
                        .lineLimit(2...4)
                }
            }
            .navigationTitle("mobile_driver.ui.request_early_complete")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("warehouse_portal.cycle_counts.text.submit") {
                        onConfirm(selectedReason, note)
                    }
                    .foregroundStyle(LabTheme.destructive)
                    .fontWeight(.semibold)
                }
            }
        }
    }
}
