import SwiftUI

struct SectionsView: View {
    @State private var items: [SectionRowIOS] = []
    @State private var name = "Dairy"
    @State private var aisle = ""
    @State private var selectedId: String?
    @State private var skuText = ""
    @State private var banner: String?
    @State private var saveError: String?
    @State private var busy = false
    private let api = APIClient.shared

    var body: some View {
        List {
            if let saveError {
                Section { Text(saveError).font(.caption).foregroundStyle(AppTheme.destructive) }
            } else if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("New section") {
                TextField("retailer_desktop.pos.text.name", text: $name)
                TextField("retailer_desktop.sections.text.aisle_tag", text: $aisle)
                Button(busy ? "…" : "Create") { Task { await create() } }.disabled(busy)
            }
            Section("portal.nav.sections") {
                if items.isEmpty {
                    Text("mobile_retailer.ui.none_yet").foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(items) { row in
                        Button {
                            selectedId = row.id
                        } label: {
                            HStack {
                                Text(row.name)
                                Spacer()
                                if selectedId == row.id { Image(systemName: "checkmark") }
                            }
                        }
                    }
                }
            }
            if selectedId != nil {
                Section("Map SKUs") {
                    TextField("mobile_retailer.ui.sku_list_comma_separated", text: $skuText)
                    Button("mobile_retailer.ui.save_skus") { Task { await saveSkus() } }.disabled(busy)
                }
            }
        }
        .navigationTitle("portal.nav.sections")
        .navigationBarTitleDisplayMode(.inline)
        .task { await refresh() }
    }

    private func refresh() async {
        do {
            let res = try await api.getSections()
            items = res.items.map { SectionRowIOS(id: $0.sectionId, name: $0.name) }
        } catch {
            banner = error.localizedDescription
        }
    }

    private func create() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createSection(name: name, aisleTag: aisle.isEmpty ? nil : aisle)
            saveError = nil
            banner = "Created"
            await refresh()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func saveSkus() async {
        guard let selectedId else { return }
        busy = true
        defer { busy = false }
        do {
            let skus = skuText.split { $0 == "," || $0.isWhitespace }.map(String.init).filter { !$0.isEmpty }
            _ = try await api.putSectionSkus(sectionId: selectedId, skus: skus)
            saveError = nil
            banner = "SKUs saved"
        } catch {
            saveError = "section_skus_failed"
        }
    }
}

struct SectionRowIOS: Identifiable {
    let id: String
    let name: String
}
