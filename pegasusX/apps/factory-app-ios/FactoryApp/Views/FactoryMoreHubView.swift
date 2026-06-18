import SwiftUI

struct FactoryMoreHubView: View {
    var onSelect: (FactorySection) -> Void

    var body: some View {
        List {
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
