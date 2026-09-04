import SwiftUI

struct StaffDetailView: View {
    let staffId: String

    @State private var staff: StaffMember?
    @State private var loading = true
    @State private var error: String?
    @State private var pin = ""
    @State private var setMsg: String?
    @State private var setting = false

    var body: some View {
        Group {
            if loading {
                ProgressView()
            } else if let error {
                ContentUnavailableView {
                    Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { Task { await load() } }
                }
            } else if let staff {
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
                        SecureField("PIN (min 4)", text: $pin)
                        Button(setting ? "Saving…" : "Set password") {
                            Task { await setPassword() }
                        }
                        .disabled(setting || pin.trimmingCharacters(in: .whitespaces).count < 4)
                        if let setMsg { Text(setMsg) }
                    }
                }
            }
        }
        .navigationTitle("portal.nav.staff")
        .task(id: staffId) { await load() }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        do {
            staff = try await FactoryService.staffDetail(id: staffId)
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }

    @MainActor
    private func setPassword() async {
        setting = true
        setMsg = nil
        do {
            try await FactoryService.setStaffPassword(id: staffId, pin: pin)
            setMsg = "Password set"
            pin = ""
        } catch {
            setMsg = error.localizedDescription
        }
        setting = false
    }
}
