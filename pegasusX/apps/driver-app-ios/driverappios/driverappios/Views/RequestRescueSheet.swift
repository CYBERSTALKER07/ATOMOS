import SwiftUI

struct RequestRescueSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = RequestRescueViewModel()

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                Section(header: Text("mobile_driver.ui.rescue_reason")) {
                    Picker("Reason", selection: $vm.reason) {
                        ForEach(RequestRescueViewModel.availableReasons, id: \.self) {
                            Text($0)
                        }
                    }
                    .pickerStyle(.menu)
                }

                Section(header: Text("mobile_driver.ui.additional_notes")) {
                    TextField("mobile_driver.ui.optional_details", text: $vm.note)
                }

                if let error = vm.errorMessage {
                    Section {
                        Text(error)
                            .font(.footnote)
                            .foregroundStyle(LabTheme.destructive)
                    }
                }

                Button {
                    Task {
                        if await vm.submitRescue() {
                            dismiss()
                        }
                    }
                } label: {
                    if vm.isSubmitting {
                        ProgressView()
                            .tint(.white)
                            .frame(maxWidth: .infinity, minHeight: 44)
                    } else {
                        Text("mobile_driver.ui.request_rescue")
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .foregroundStyle(.white)
                    }
                }
                .padding()
                .background(LabTheme.warning)
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                .disabled(vm.isSubmitting)
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_driver.ui.request_rescue")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                        .frame(minWidth: 44, minHeight: 44)
                }
            }
        }
    }
}
