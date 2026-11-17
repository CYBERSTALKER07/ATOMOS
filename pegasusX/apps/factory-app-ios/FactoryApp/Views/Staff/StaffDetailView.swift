import SwiftUI

struct StaffDetailView: View {
    let staffId: String
    @State private var vm: StaffDetailViewModel

    init(staffId: String) {
        self.staffId = staffId
        _vm = State(initialValue: StaffDetailViewModel(staffId: staffId))
    }

    var body: some View {
        @Bindable var vm = vm

        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.selection()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            } else if let staff = vm.staff {
                ResponsiveGridContentWrapper {
                    Section {
                        LabeledContent("Name", value: staff.name)
                        LabeledContent("Role", value: staff.role)
                        LabeledContent("Staff ID", value: staff.id)
                        LabeledContent("Phone", value: staff.phone.isEmpty ? "—" : staff.phone)
                        LabeledContent("Status", value: staff.status.isEmpty ? "ACTIVE" : staff.status)
                        LabeledContent("Joined", value: staff.joinedAt.isEmpty ? "—" : staff.joinedAt)
                    }
                    Section("Set login PIN") {
                        SecureField("PIN (min 4)", text: $vm.pin)
                            .frame(minHeight: 44)
                            .accessibilityLabel("Login PIN, minimum four characters")
                        Button(vm.setting ? "Saving…" : "Set password") {
                            Task { await vm.setPassword() }
                        }
                        .frame(minWidth: 44, minHeight: 44, alignment: .leading)
                        .disabled(!vm.canSetPassword)
                        .accessibilityLabel("Save staff login PIN")
                        if let setMsg = vm.setMsg {
                            Text(setMsg)
                                .font(.caption)
                                .foregroundStyle(setMsg == "Password set" ? .green : .red)
                                .accessibilityLabel(setMsg)
                        }
                    }
                }
            }
        }
        .navigationTitle("portal.nav.staff")
        .task(id: staffId) {
            await vm.load()
        }
    }
}
