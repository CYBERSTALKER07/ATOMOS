import SwiftUI

struct SupplierIdentityCard: View {
    let profile: SupplierProfile

    var body: some View {
        Group {
            Section("Business") {
                LabeledContent("Legal name", value: profile.legalName)
                LabeledContent("Contact", value: profile.contactName)
                LabeledContent("Email", value: profile.email)
                LabeledContent("Phone", value: profile.phone)
                LabeledContent("Country", value: profile.country)
                LabeledContent("Currency", value: profile.currency)
            }
            Section("Status") {
                LabeledContent("Registered", value: profile.isRegistered ? "Yes" : "No")
                LabeledContent("Configured", value: profile.isConfigured ? "Yes" : "No")
                if !profile.selectedGateways.isEmpty {
                    LabeledContent("Gateways", value: profile.selectedGateways.joined(separator: ", "))
                }
            }
            if !profile.categories.isEmpty {
                Section("Categories") {
                    ForEach(profile.categories, id: \.self) { Text($0) }
                }
            }
        }
    }
}
