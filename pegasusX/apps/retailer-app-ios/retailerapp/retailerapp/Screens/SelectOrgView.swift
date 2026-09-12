import SwiftUI

/// C1.3 multi-org picker after PendingOrgSelect login.
struct SelectOrgView: View {
    @Bindable var auth: AuthManager
    @State private var busyId: String?

    var body: some View {
        VStack(spacing: 20) {
            Text("retailer_desktop.auth.select_org.text.choose_organization")
                .font(.title2.weight(.semibold))
            Text("mobile_retailer.ui.your_phone_is_linked_to_more_than_one_retailer")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)

            if let memberships = auth.pendingMemberships, !memberships.isEmpty {
                List(memberships) { m in
                    Button {
                        Task {
                            busyId = m.retailerId
                            await auth.selectOrg(retailerId: m.retailerId)
                            busyId = nil
                        }
                    } label: {
                        HStack {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(m.name?.isEmpty == false ? m.name! : m.retailerId)
                                    .font(.headline)
                                Text(m.retailerRole)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            if busyId == m.retailerId {
                                ProgressView()
                            }
                        }
                    }
                    .disabled(auth.isLoading)
                }
                .listStyle(.plain)
            } else {
                Text("mobile_retailer.ui.no_organizations_available")
                    .foregroundStyle(.secondary)
            }

            if let err = auth.errorMessage {
                Text(err)
                    .font(.footnote)
                    .foregroundStyle(.red)
                    .multilineTextAlignment(.center)
            }

            Button("mobile_retailer.ui.back_to_sign_in") {
                auth.logout()
            }
            .font(.subheadline)
        }
        .padding()
    }
}
