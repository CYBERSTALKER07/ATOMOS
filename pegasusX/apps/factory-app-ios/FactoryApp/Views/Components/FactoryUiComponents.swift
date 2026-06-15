import SwiftUI

struct KpiTile: View {
    let title: String
    let value: String
    let systemImage: String
    let tint: Color
    var supporting: String? = nil
    var chip: (text: String, tint: Color)? = nil
    var staggerIndex: Int? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                Image(systemName: systemImage)
                    .foregroundStyle(tint)
                Spacer()
                if let chip {
                    Text(chip.text)
                        .font(.caption2.bold())
                        .padding(.horizontal, LabTheme.spacingSM)
                        .padding(.vertical, LabTheme.spacingXS)
                        .foregroundStyle(chip.tint)
                        .background(chip.tint.opacity(0.14), in: Capsule())
                }
            }
            Text(value)
                .font(.title2.bold())
                .minimumScaleFactor(0.8)
                .lineLimit(1)
            Text(title)
                .font(.subheadline.bold())
            if let supporting, !supporting.isEmpty {
                Text(supporting)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .modifier(OptionalStaggerModifier(index: staggerIndex))
    }
}

private struct OptionalStaggerModifier: ViewModifier {
    let index: Int?

    func body(content: Content) -> some View {
        if let index {
            content.staggeredAppear(index: index)
        } else {
            content
        }
    }
}

struct FactorySectionHeader: View {
    let title: String
    var subtitle: String? = nil
    var count: Int? = nil

    var body: some View {
        HStack(alignment: .firstTextBaseline) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(title)
                    .font(.headline)
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            Spacer()
            if let count {
                Text("\(count)")
                    .font(.footnote.bold())
                    .padding(.horizontal, LabTheme.spacingSM)
                    .padding(.vertical, LabTheme.spacingXS)
                    .foregroundStyle(LabTheme.secondaryLabel)
                    .background(LabTheme.tertiaryBackground, in: Capsule())
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct FactoryStatusBadge: View {
    let text: String
    var tint: Color? = nil
    var emphasized: Bool = true

    private var resolvedTint: Color {
        tint ?? LabTheme.statusTint(for: text)
    }

    var body: some View {
        Text(text.replacingOccurrences(of: "_", with: " "))
            .font(emphasized ? .caption.bold() : .caption2.bold())
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .foregroundStyle(emphasized ? resolvedTint : LabTheme.secondaryLabel)
            .background(
                emphasized ? resolvedTint.opacity(0.14) : LabTheme.tertiaryBackground,
                in: Capsule()
            )
    }
}
