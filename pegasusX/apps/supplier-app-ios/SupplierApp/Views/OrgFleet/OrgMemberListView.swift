import SwiftUI

struct OrgMemberListView: View {
    let orgMembers: [SupplierOrgMember]
    @Binding var editingMember: SupplierOrgMember?
    @Binding var showEditMemberSheet: Bool
    @Binding var memberActionId: String?
    let deactivateAction: (String) async -> Void
    
    var body: some View {
        Group {
            if orgMembers.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_org_members", message: "mobile_supplier.empty.org_members_message")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(orgMembers) { member in
                        VStack(alignment: .leading) {
                            Text(member.name).font(.headline)
                            Text(L10n.format(
                                "mobile_supplier.chrome.member_line",
                                member.supplierRole,
                                member.phone,
                                member.isActive ? L10n.format("mobile_supplier.chrome.active") : L10n.format("mobile_supplier.chrome.inactive")
                            ))
                                .font(.caption).foregroundStyle(.secondary)
                        }
                        .contentShape(Rectangle())
                        .onTapGesture {
                            editingMember = member
                            showEditMemberSheet = true
                        }
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if member.isActive {
                                Button(role: .destructive) {
                                    Task { await deactivateAction(member.userId) }
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
