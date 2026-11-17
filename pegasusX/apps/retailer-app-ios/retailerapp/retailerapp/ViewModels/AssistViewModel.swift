import Foundation
import Observation

struct AssistSectionOption: Identifiable, Hashable {
    let id: String
    let name: String
}

struct AssistRowIOS: Identifiable {
    let id: String
    let note: String
    let status: String
}

@Observable
@MainActor
final class AssistViewModel {
    var sections: [AssistSectionOption] = []
    var selectedSectionId = ""
    var note = ""
    var tickets: [AssistRowIOS] = []
    var banner: String?
    var busy = false

    private let api = APIClient.shared

    var canCreate: Bool {
        !busy && !note.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !selectedSectionId.isEmpty
    }

    func refresh() async {
        do {
            let secs = try await api.getSections()
            sections = secs.items.map { AssistSectionOption(id: $0.sectionId, name: $0.name) }
            if selectedSectionId.isEmpty || !sections.contains(where: { $0.id == selectedSectionId }) {
                selectedSectionId = sections.first?.id ?? ""
            }
            let list = try await api.getAssistTickets()
            tickets = list.items.map {
                AssistRowIOS(id: $0.ticketId, note: $0.note, status: $0.status)
            }
        } catch {
            banner = error.localizedDescription
        }
    }

    func create() async {
        guard !selectedSectionId.isEmpty else { return }
        busy = true
        defer { busy = false }
        do {
            _ = try await api.createAssistTicket(
                sectionId: selectedSectionId,
                note: note.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            note = ""
            banner = "Opened"
            Haptics.success()
            await refresh()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }

    func act(id: String, action: String) async {
        busy = true
        defer { busy = false }
        do {
            if action == "claim" {
                _ = try await api.claimAssistTicket(ticketId: id)
            } else {
                _ = try await api.completeAssistTicket(ticketId: id)
            }
            Haptics.success()
            await refresh()
        } catch {
            banner = error.localizedDescription
            Haptics.error()
        }
    }
}
