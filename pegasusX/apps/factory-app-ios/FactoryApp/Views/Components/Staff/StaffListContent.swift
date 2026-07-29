import SwiftUI

struct StaffListContent: View {
    let staff: [StaffMember]

    var body: some View {
        ResponsiveGridContentWrapper {
            Section {
                FactorySectionHeader(
                    title: "Staff roster",
                    subtitle: "\(staff.count) operators on record"
                )
                .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                .listRowBackground(Color.clear)
            }

            Section {
                ForEach(Array(staff.enumerated()), id: \.element.id) { index, member in
                    NavigationLink(value: member.id) {
                        StaffRow(member: member)
                    }
                    .staggeredAppear(index: index)
                }
            }
        }
    }
}
