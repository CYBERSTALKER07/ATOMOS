import SwiftUI

struct FactoryMoreHubView: View {
    var body: some View {
        ResponsiveGridContentWrapper {
            let extraPrimary = FactorySection.primarySections.filter { !FactorySection.compactTabs.contains($0) }
            if !extraPrimary.isEmpty {
                Section("Primary") {
                    ForEach(extraPrimary) { section in
                        sectionLink(section)
                    }
                }
            }
            Section("Operations") {
                ForEach(FactorySection.operationsSections) { section in
                    sectionLink(section)
                }
            }
            Section("Intelligence") {
                ForEach(FactorySection.intelligenceSections) { section in
                    sectionLink(section)
                }
            }
        }
        .navigationTitle("mobile_factory.ui.more")
    }

    private func sectionLink(_ section: FactorySection) -> some View {
        NavigationLink(value: section) {
            Label(section.rawValue, systemImage: section.icon)
                .frame(minHeight: 44, alignment: .leading)
        }
        .accessibilityLabel(section.rawValue)
    }
}
