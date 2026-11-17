import SwiftUI

struct CreateStaffSheet: View {
    @Bindable var vm: StaffViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $vm.newStaffName)
                    .frame(minHeight: 44)
                    .accessibilityLabel("Staff member name")
                TextField("warehouse_portal.staff.text.role", text: $vm.newStaffRole)
                    .frame(minHeight: 44)
                    .accessibilityLabel("Staff member role")
                if let error = vm.createError {
                    Text(error)
                        .foregroundStyle(.red)
                        .font(.caption)
                        .accessibilityLabel("Error: \(error)")
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_factory.ui.add_staff")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button {
                        PegasusHaptics.selection()
                        dismiss()
                    } label: {
                        Text("common.action.cancel")
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Cancel creating staff")
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        Task {
                            if await vm.createStaff() {
                                dismiss()
                            }
                        }
                    } label: {
                        Text("mobile_factory.ui.create")
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .disabled(!vm.canSubmitCreate)
                    .accessibilityLabel("Create staff member")
                }
            }
        }
    }
}
