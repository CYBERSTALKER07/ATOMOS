import SwiftUI

struct StaffDetailView: View {
    let staffId: String

    @State private var staff: StaffMember?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
            } else if let error {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { Task { await load() } }
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
                }
            }
        }
        .navigationTitle("Staff")
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
}
