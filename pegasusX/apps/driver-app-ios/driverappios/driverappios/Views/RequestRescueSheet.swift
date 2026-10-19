import SwiftUI

struct RequestRescueSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var reason: String = "Engine Failure"
    @State private var note: String = ""
    @State private var isSubmitting = false
    
    let reasons = ["Engine Failure", "Flat Tire", "Accident", "Other"]
    
    var body: some View {
        NavigationStack {
            Form {
                Section(header: Text("mobile_driver.ui.rescue_reason")) {
                    Picker("Reason", selection: $reason) {
                        ForEach(reasons, id: \.self) {
                            Text($0)
                        }
                    }
                    .pickerStyle(.menu)
                }
                
                Section(header: Text("mobile_driver.ui.additional_notes")) {
                    TextField("mobile_driver.ui.optional_details", text: $note)
                }
                
                Button(action: submitRescue) {
                    if isSubmitting {
                        ProgressView()
                    } else {
                        Text("mobile_driver.ui.request_rescue")
                            .frame(maxWidth: .infinity)
                            .foregroundStyle(.white)
                    }
                }
                .padding()
                .background(LabTheme.warning)
                .cornerRadius(10)
                .disabled(isSubmitting)
            }
            .navigationTitle("mobile_driver.ui.request_rescue")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
            }
        }
    }
    
    private func submitRescue() {
        isSubmitting = true
        Task {
            do {
                try await FleetServiceLive.shared.requestRescue(reason: reason, note: note)
                dismiss()
            } catch {
                print("Failed to request rescue: \(error)")
            }
            isSubmitting = false
        }
    }
}
