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
                Section(header: Text("Rescue Reason")) {
                    Picker("Reason", selection: $reason) {
                        ForEach(reasons, id: \.self) {
                            Text($0)
                        }
                    }
                    .pickerStyle(.menu)
                }
                
                Section(header: Text("Additional Notes")) {
                    TextField("Optional details...", text: $note)
                }
                
                Button(action: submitRescue) {
                    if isSubmitting {
                        ProgressView()
                    } else {
                        Text("Request Rescue")
                            .frame(maxWidth: .infinity)
                            .foregroundStyle(.white)
                    }
                }
                .padding()
                .background(LabTheme.warning)
                .cornerRadius(10)
                .disabled(isSubmitting)
            }
            .navigationTitle("Request Rescue")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
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
