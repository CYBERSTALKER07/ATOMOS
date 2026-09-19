import SwiftUI

struct TreasuryTransactionList: View {
    let invoices: [Invoice]
    
    var body: some View {
        if invoices.isEmpty {
            WarehouseEmptyView(title: "No Invoices", message: "No invoices found for this warehouse.")
        } else {
            ResponsiveGridContentWrapper {
                ForEach(invoices) { inv in
                    HStack(alignment: .top) {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(inv.retailerName)
                                .font(.headline)
                            Text(L10n.format("mobile_warehouse.ui.formatted_currency_due_duedate", "\(inv.amountUzs.formatted())", "\(inv.currency)", "\(inv.dueDate)"))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                            let ownerType = inv.payoutOwnerType.isEmpty ? "SUPPLIER" : inv.payoutOwnerType
                            let ownerID = inv.payoutOwnerId.isEmpty ? "" : String(inv.payoutOwnerId.prefix(8))
                            Text("Owner \(ownerType)\(ownerID.isEmpty ? "" : ":\(ownerID)") · Fee \(inv.feeAmount.formatted()) · Net \(inv.netPayoutAmount.formatted())")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        WarehouseStatusBadge(text: inv.status)
                    }
                }
            }
        }
    }
}
