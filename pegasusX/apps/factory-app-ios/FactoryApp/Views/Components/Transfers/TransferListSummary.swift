import SwiftUI

struct TransferListSummary: View {
    let count: Int
    let selectedFilter: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(L10n.format("mobile_factory.ui.count_transfers_in_view_3", "\(count)"))
                .font(.headline)
            Text(selectedFilter == "ALL" ? "Showing every transfer state across the factory queue." : "Filtered to \(selectedFilter) transfers.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}
