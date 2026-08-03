import SwiftUI

struct TodaySummaryCard: View {
    var vm: FleetViewModel

    var body: some View {
        VStack(spacing: LabTheme.s12) {
            DriverSectionHeader(title: "Today", trailing: todayDate)

            HStack(spacing: LabTheme.s12) {
                KpiTile(label: "Pending", value: "\(vm.pendingMissions.count)", icon: "clock")
                KpiTile(label: "Done", value: "\(vm.completedIds.count)", icon: "checkmark", tint: LabTheme.success)
                KpiTile(label: "Revenue", value: totalRevenue, icon: "banknote")
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
