import SwiftUI

struct EntityResolutionView: View {
    @State private var vm = EntityResolutionViewModel()

    var body: some View {
        @Bindable var vm = vm
        Form {
            Picker("mobile_supplier.chrome.type", selection: $vm.entityType) {
                ForEach(EntityResolutionViewModel.types, id: \.self) { Text($0).tag($0) }
            }
            TextField("mobile_supplier.chrome.query", text: $vm.query)
            TextField("mobile_supplier.chrome.entity_id", text: $vm.entityId)
            Button(LocalizedStringKey(vm.busy ? "mobile_supplier.chrome.resolving" : "mobile_supplier.chrome.resolve")) {
                PegasusHaptics.light()
                Task { await vm.runResolve() }
            }
            .frame(minHeight: 44)
            .disabled(vm.busy)
            if let error = vm.error {
                Text(error).foregroundStyle(SupplierTheme.destructive)
            }
            if let resolved = vm.resolved {
                Text(L10n.format("mobile_supplier.chrome.requested_candidates", resolved.requestedType, "\(resolved.candidates.count)"))
                    .font(.footnote)
                if let top = resolved.resolved {
                    Text(L10n.format("mobile_supplier.chrome.top_candidate", top.label, top.entityId, "\(top.score)"))
                        .font(.caption)
                } else {
                    Text("mobile_supplier.chrome.no_deterministic_match").font(.caption)
                }
                ForEach(resolved.candidates) { c in
                    Button("\(c.label) · \(c.entityType)/\(c.entityId) · \(c.confidenceClass)") {
                        PegasusHaptics.light()
                        Task { await vm.runExplain(type: c.entityType, id: c.entityId) }
                    }
                    .frame(minHeight: 44)
                }
            }
            if let explain = vm.explain {
                Text(L10n.format("mobile_supplier.chrome.lineage_for", explain.source.label)).font(.headline)
                ForEach(Array(explain.projection.edges.enumerated()), id: \.offset) { _, e in
                    Text("\(e.relation): \(e.from) → \(e.to)").font(.caption)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.entity_resolution")
    }
}
