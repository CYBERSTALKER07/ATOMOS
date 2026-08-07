import SwiftUI

struct OrgMemberListView: View {
    let orgMembers: [SupplierOrgMember]
    let onEdit: (SupplierOrgMember) -> Void
    let onDeactivate: (String) async -> Void
    let memberActionId: String?
    
    var body: some View {
        Group {
            if orgMembers.isEmpty {
                SupplierEmptyView(title: "No org members", message: "Create warehouse, factory, or payload staff.")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(orgMembers) { member in
                        VStack(alignment: .leading) {
                            Text(member.name).font(.headline)
                            Text("\(member.supplierRole) · \(member.phone) · \(member.isActive ? "Active" : "Inactive")")
                                .font(.caption).foregroundStyle(.secondary)
                        }
                        .contentShape(Rectangle())
                        .onTapGesture {
                            onEdit(member)
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if member.isActive {
                                Button(role: .destructive) {
                                    Task { await onDeactivate(member.userId) }
                                } label: {
                                    Text("supplier_portal.demand.signals.text.deactivate")
                                }
                                .disabled(memberActionId == member.userId)
                            }
                        }
                    }
                }
            }
        }
    }
}
