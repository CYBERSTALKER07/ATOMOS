import SwiftUI

struct KpiGrid: View {
    let activeOrdersCount: Int
    let predictionsCount: Int
    let reorderProductsCount: Int
    let horizontalSizeClass: UserInterfaceSizeClass?

    private var kpiGridMin: CGFloat {
        horizontalSizeClass == .regular ? 160 : 140
    }

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "At a glance", subtitle: "Live retailer KPIs")

            LazyVGrid(
                columns: [GridItem(.adaptive(minimum: kpiGridMin), spacing: AppTheme.spacingMD)],
                spacing: AppTheme.spacingMD
            ) {
                NavigationLink {
                    OrdersView()
                } label: {
                    KpiTile(
                        title: "Active Orders",
                        value: "\(activeOrdersCount)",
                        systemImage: "shippingbox.fill",
                        tint: AppTheme.accent,
                        chip: activeOrdersCount == 0 ? nil : ("LIVE", AppTheme.success)
                    )
                }
                .buttonStyle(.pressable)

                NavigationLink {
                    FutureDemandView()
                } label: {
                    KpiTile(
                        title: "Reorder suggestions",
                        value: "\(predictionsCount)",
                        systemImage: "sparkles",
                        tint: AppTheme.info,
                        chip: predictionsCount == 0 ? nil : ("NEW", AppTheme.warning)
                    )
                }
                .buttonStyle(.pressable)

                NavigationLink {
                    ProcurementView()
                } label: {
                    KpiTile(
                        title: "Quick Reorder",
                        value: "\(reorderProductsCount)",
                        systemImage: "arrow.clockwise",
                        tint: AppTheme.success
                    )
                }
                .buttonStyle(.pressable)
            }
        }
    }
}
