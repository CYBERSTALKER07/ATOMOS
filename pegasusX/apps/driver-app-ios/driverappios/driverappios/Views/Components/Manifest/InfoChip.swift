import SwiftUI

struct InfoChip: View {
    let icon: String
    let text: String
    
    var body: some View {
        HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 10, weight: .semibold))
            Text(text)
                .font(.system(size: 12, weight: .medium))
        }
        .foregroundStyle(LabTheme.fgSecondary)
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(LabTheme.fg.opacity(0.04), in: Capsule())
    }
}
