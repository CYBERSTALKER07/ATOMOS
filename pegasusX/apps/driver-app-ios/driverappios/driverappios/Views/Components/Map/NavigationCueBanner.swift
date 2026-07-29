import SwiftUI

struct NavigationCueBanner: View {
    let cue: NavigationCue
    
    var body: some View {
        HStack(alignment: .center, spacing: 10) {
            Image(systemName: "location.north.line.fill")
                .font(.system(size: 14, weight: .bold))
                .foregroundStyle(LabTheme.fg)
            VStack(alignment: .leading, spacing: 2) {
                Text(RouteNavigation.formatDistance(cue.distanceM))
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                Text(cue.instruction)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(LabTheme.fg.opacity(0.85))
                    .lineLimit(2)
            }
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay(RoundedRectangle(cornerRadius: 16, style: .continuous).stroke(LabTheme.separator, lineWidth: 0.5))
    }
}
