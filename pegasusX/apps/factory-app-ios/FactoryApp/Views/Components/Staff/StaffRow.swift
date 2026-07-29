import SwiftUI

struct StaffRow: View {
    let member: StaffMember

    var body: some View {
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
            FactoryStatusBadge(text: member.status)
        }
    }
}
