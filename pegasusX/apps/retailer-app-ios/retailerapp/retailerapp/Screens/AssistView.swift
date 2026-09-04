import SwiftUI

struct AssistView: View {
    @State private var sections: [(id: String, name: String)] = []
    @State private var sectionId: String?
    @State private var note = ""
    @State private var tickets: [AssistRowIOS] = []
    @State private var banner: String?
    @State private var busy = false
    private let api = APIClient.shared

    var body: some View {
        List {
            if let banner { Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) } }
            Section("New ticket") {
                if sections.isEmpty {
                    Text("retailer_desktop.assist.text.create_a_section_first").foregroundStyle(AppTheme.textTertiary)
                } else {
                    Picker("Section", selection: Binding(
                        get: { sectionId ?? sections.first?.id ?? "" },
                        set: { sectionId = $0 }
                    )) {
                        ForEach(sections, id: \.id) { s in
                            Text(s.name).tag(s.id)
                        }
                    }
                }
                TextField("mobile_retailer.ui.help_note", text: $note)
                Button(busy ? "…" : "Open ticket") { Task { await create() } }
                    .disabled(busy || note.isEmpty || sectionId == nil)
            }
            Section("Queue") {
                if tickets.isEmpty {
                    Text("mobile_retailer.ui.no_tickets").foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(tickets) { t in
                        VStack(alignment: .leading, spacing: 6) {
                            Text(L10n.format("mobile_retailer.ui.status_note", "\(t.status)", "\(t.note)"))
                            HStack {
                                if t.status == "OPEN" {
                                    Button("mobile_retailer.ui.claim") { Task { await act(id: t.id, action: "claim") } }
                                }
                                if t.status == "OPEN" || t.status == "CLAIMED" {
                                    Button("mobile_retailer.ui.complete") { Task { await act(id: t.id, action: "complete") } }
                                }
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("retailer_desktop.assist.text.floor_assist")
        .navigationBarTitleDisplayMode(.inline)
        .task { await refresh() }
    }

    private func refresh() async {
        do {
            let secs = try await api.getSections()
            sections = secs.items.map { ($0.sectionId, $0.name) }
            if sectionId == nil { sectionId = sections.first?.id }
            let list = try await api.getAssistTickets()
            tickets = list.items.map {
                AssistRowIOS(id: $0.ticketId, note: $0.note, status: $0.status)
            }
        } catch {
            banner = error.localizedDescription
        }
    }

    private func create() async {
        guard let sectionId else { return }
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createAssistTicket(sectionId: sectionId, note: note)
            note = ""
            banner = "Opened"
            await refresh()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func act(id: String, action: String) async {
        do {
            if action == "claim" {
                _ = try await api.claimAssistTicket(ticketId: id)
            } else {
                _ = try await api.completeAssistTicket(ticketId: id)
            }
            await refresh()
        } catch {
            banner = error.localizedDescription
        }
    }
}

struct AssistRowIOS: Identifiable {
    let id: String
    let note: String
    let status: String
}
