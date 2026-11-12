import SwiftUI

struct SupplierIdentityCard: View {
    let profile: SupplierProfile

    var body: some View {
        Group {
            Section("mobile_supplier.chrome.business") {
                LabeledContent("mobile_supplier.chrome.legal_name", value: profile.legalName)
                LabeledContent("mobile_supplier.chrome.contact", value: profile.contactName)
                LabeledContent("mobile_supplier.chrome.email", value: profile.email)
                LabeledContent("mobile_supplier.chrome.phone", value: profile.phone)
                LabeledContent("mobile_supplier.chrome.country", value: profile.country)
                LabeledContent("mobile_supplier.chrome.currency", value: profile.currency)
            }
            Section("mobile_supplier.chrome.status") {
                LabeledContent("mobile_supplier.chrome.registered", value: profile.isRegistered ? L10n.format("mobile_supplier.chrome.yes") : L10n.format("mobile_supplier.chrome.no"))
                LabeledContent("mobile_supplier.chrome.configured", value: profile.isConfigured ? L10n.format("mobile_supplier.chrome.yes") : L10n.format("mobile_supplier.chrome.no"))
                if !profile.selectedGateways.isEmpty {
                    LabeledContent("mobile_supplier.chrome.gateways", value: profile.selectedGateways.joined(separator: ", "))
                }
            }
            if !profile.categories.isEmpty {
                Section("mobile_supplier.chrome.categories") {
                    ForEach(profile.categories, id: \.self) { Text($0) }
                }
            }
        }
    }
}
