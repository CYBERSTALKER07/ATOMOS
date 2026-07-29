import SwiftUI

struct StaffList: View {
    let staff: [StaffMember]
    let loading: Bool
    let error: String?
    let onRetry: () -> Void

    var body: some View {
        Group {
            if loading && staff.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error, staff.isEmpty {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { onRetry() }
                }
            } else if staff.isEmpty {
                ContentUnavailableView("No Staff", systemImage: "person.2", description: Text("Add staff members"))
            } else {
                ResponsiveGridView(data: staff) { member in
                    HStack {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(member.name)
                                .font(.headline)
                            Text("\(member.role) · \(member.phone)")
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        Text(member.isActive ? "Active" : "Inactive")
                            .font(.caption.bold())
                            .padding(.horizontal, LabTheme.spacingSM)
                            .padding(.vertical, LabTheme.spacingXS)
                            .foregroundStyle(member.isActive ? Color.primary : Color.white)
                            .background(member.isActive ? AnyShapeStyle(Color.clear) : AnyShapeStyle(Color.red), in: Capsule())
                            .overlay {
                                if member.isActive {
                                    Capsule().strokeBorder(Color.gray.opacity(0.3))
                                }
                            }
                    }
                }
            }
        }
    }
}
