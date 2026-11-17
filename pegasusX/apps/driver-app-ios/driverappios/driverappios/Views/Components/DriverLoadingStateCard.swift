import SwiftUI

struct DriverLoadingStateCard: View {
    let title: String
    let message: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.s16) {
            HStack(spacing: 12) {
                ProgressView()
                    .controlSize(.large)
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.headline)
                        .foregroundStyle(LabTheme.fg)
                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(LabTheme.fgSecondary)
                }
            }

            VStack(spacing: 10) {
                DriverStateSkeletonBar(width: nil)
                DriverStateSkeletonBar(width: nil)
                DriverStateSkeletonBar(width: 180)
            }
        }
        .padding(24)
        .frame(maxWidth: 420)
        .labCard()
        .staggeredAppear(index: 0)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title). \(message)")
    }
}

private struct DriverStateSkeletonBar: View {
    let width: CGFloat?

    var body: some View {
        RoundedRectangle(cornerRadius: 10, style: .continuous)
            .fill(LabTheme.separator.opacity(0.14))
            .frame(maxWidth: width ?? .infinity)
            .frame(height: 14)
            .shimmer()
            .accessibilityHidden(true)
    }
}
