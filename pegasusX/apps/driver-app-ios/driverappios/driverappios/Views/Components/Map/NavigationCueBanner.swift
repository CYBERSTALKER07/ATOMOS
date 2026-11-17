import SwiftUI

struct NavigationCueBanner: View {
    let cue: NavigationCue
    
    var body: some View {
        HStack(alignment: .center, spacing: 10) {
            Image(systemName: "location.north.line.fill")
                .font(.system(.subheadline, weight: .bold))
                .foregroundStyle(LabTheme.fg)
            VStack(alignment: .leading, spacing: 2) {
                Text(RouteNavigation.formatDistance(cue.distanceM))
                    .font(.system(.footnote, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                Text(cue.instruction)
                    .font(.system(.subheadline, weight: .medium))
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
