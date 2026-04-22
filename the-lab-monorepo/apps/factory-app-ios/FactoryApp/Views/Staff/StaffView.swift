import SwiftUI

struct StaffView: View {
    @State private var staff: [StaffMember] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if staff.isEmpty {
                    ContentUnavailableView("No Staff", systemImage: "person.2", description: Text("No staff members found"))
                } else {
                    List {
                        ForEach(Array(staff.enumerated()), id: \.element.id) { index, member in
                            HStack(spacing: LabTheme.spacingLG) {
                                Image(systemName: "person.circle")
                                    .font(.title2)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 32)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(member.name)
                                        .font(.subheadline.bold())
                                    Text(member.phone)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Text(member.role)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Text(member.status)
                                    .font(.caption2.bold())
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 3)
                                    .background(.quaternary)
                                    .clipShape(Capsule())
                            }
                            .staggeredAppear(index: index)
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Staff")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .task { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                staff = try await FactoryService.staff().staff
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
