import SwiftUI

struct AssistView: View {
    @State private var vm = AssistViewModel()

    var body: some View {
        @Bindable var vm = vm
        List {
            if let banner = vm.banner {
                Section {
                    Text(banner)
                        .font(.caption)
                        .foregroundStyle(AppTheme.accent)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel(banner)
                }
            }
            Section("New ticket") {
                if vm.sections.isEmpty {
                    Text("retailer_desktop.assist.text.create_a_section_first")
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    Picker("Section", selection: $vm.selectedSectionId) {
                        ForEach(vm.sections) { section in
                            Text(section.name).tag(section.id)
                        }
                    }
                    .accessibilityLabel("Assist section")
                }
                TextField("mobile_retailer.ui.help_note", text: $vm.note)
                    .accessibilityLabel("Help note")
                Button(vm.busy ? "…" : "Open ticket") {
                    Task { await vm.create() }
                }
                .pegasusMinimumHitTarget()
                .disabled(!vm.canCreate)
                .accessibilityLabel("Open assist ticket")
            }
            Section("Queue") {
                if vm.tickets.isEmpty {
                    Text("mobile_retailer.ui.no_tickets")
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(vm.tickets) { ticket in
                        VStack(alignment: .leading, spacing: 6) {
                            Text(L10n.format("mobile_retailer.ui.status_note", "\(ticket.status)", "\(ticket.note)"))
                                .font(.system(.subheadline, design: .rounded))
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("Ticket \(ticket.status), \(ticket.note)")
                            HStack(spacing: AppTheme.spacingSM) {
                                if ticket.status == "OPEN" {
                                    Button("mobile_retailer.ui.claim") {
                                        Task { await vm.act(id: ticket.id, action: "claim") }
                                    }
                                    .buttonStyle(.bordered)
                                    .pegasusMinimumHitTarget()
                                    .disabled(vm.busy)
                                    .accessibilityLabel("Claim ticket \(ticket.note)")
                                }
                                if ticket.status == "OPEN" || ticket.status == "CLAIMED" {
                                    Button("mobile_retailer.ui.complete") {
                                        Task { await vm.act(id: ticket.id, action: "complete") }
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .pegasusMinimumHitTarget()
                                    .disabled(vm.busy)
                                    .accessibilityLabel("Complete ticket \(ticket.note)")
                                }
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("retailer_desktop.assist.text.floor_assist")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.refresh() }
        .refreshable { await vm.refresh() }
    }
}
