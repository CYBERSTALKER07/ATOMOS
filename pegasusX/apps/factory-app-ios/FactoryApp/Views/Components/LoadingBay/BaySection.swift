import SwiftUI

struct BaySection: View {
    let title: String
    let count: Int
    let transfers: [Transfer]
    let emptyMessage: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            FactorySectionHeader(title: title, count: count)

            if transfers.isEmpty {
                Text(emptyMessage)
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(LabTheme.spacingLG)
                    .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            } else {
                LazyVStack(spacing: LabTheme.spacingSM) {
                    ForEach(Array(transfers.enumerated()), id: \.element.id) { index, transfer in
                        BayTransferCard(transfer: transfer)
                            .staggeredAppear(index: index)
                    }
                }
            }
        }
    }
}
