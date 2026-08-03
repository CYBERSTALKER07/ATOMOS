import SwiftUI

struct TreasuryTransactionList: View {
    let invoices: [Invoice]
<<<<<<< HEAD

=======
    
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
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
                            Text("\(inv.amountUzs.formatted()) \(inv.currency) · Due: \(inv.dueDate)")
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
