import SwiftUI

struct TodaySummaryCard: View {
    var vm: FleetViewModel
    var onSelectPending: (() -> Void)? = nil
    var onSelectDone: (() -> Void)? = nil
    var onSelectRevenue: (() -> Void)? = nil

    var body: some View {
        VStack(spacing: LabTheme.s12) {
            DriverSectionHeader(title: "Today", trailing: todayDate)

            HStack(spacing: LabTheme.s12) {
                Button {
                    Haptics.light()
                    onSelectPending?()
                } label: {
                    KpiTile(label: "Pending", value: "\(vm.pendingMissions.count)", icon: "clock")
                }
                .buttonStyle(.plain)

                Button {
                    Haptics.light()
                    onSelectDone?()
                } label: {
                    KpiTile(label: "Done", value: "\(vm.completedIds.count)", icon: "checkmark", tint: LabTheme.success)
                }
                .buttonStyle(.plain)

                Button {
                    Haptics.light()
                    onSelectRevenue?()
                } label: {
                    KpiTile(label: "Revenue", value: totalRevenue, icon: "banknote")
                }
                .buttonStyle(.plain)
            }
        }
    }

    private var totalRevenue: String {
        let total = vm.completedMissions.reduce(0) { $0 + $1.amount }
        if total == 0 { return "—" }
        return total.formatted(.number.grouping(.automatic))
    }

    private var todayDate: String {
        Date().formatted(.dateTime.day().month(.abbreviated).year()).uppercased()
    }
}
