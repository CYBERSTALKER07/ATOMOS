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
                    Text("Remaining orders will be returned to the supplier for next-day re-dispatch.")
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
                    TextField("Add a note", text: $note, axis: .vertical)
                        .lineLimit(2...4)
                }
            }
            .navigationTitle("Request Early Complete")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") {
                        onConfirm(selectedReason, note)
                    }
                    .foregroundStyle(LabTheme.destructive)
                    .fontWeight(.semibold)
                }
            }
        }
    }
}
