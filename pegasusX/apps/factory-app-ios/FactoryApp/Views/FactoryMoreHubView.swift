import SwiftUI

struct FactoryMoreHubView: View {
    var onSelect: (FactorySection) -> Void

    var body: some View {
        ResponsiveGridContentWrapper {
            Section("Primary") {
                ForEach(FactorySection.primarySections.filter { !FactorySection.compactTabs.contains($0) }) { section in
                    Button(section.rawValue) { onSelect(section) }
                }
            }
            Section("Operations") {
                ForEach(FactorySection.operationsSections) { section in
                    Button(section.rawValue) { onSelect(section) }
                }
            }
            Section("Intelligence") {
                ForEach(FactorySection.intelligenceSections) { section in
                    Button(section.rawValue) { onSelect(section) }
                }
            }
        }
        .navigationTitle("More")
    }
}
