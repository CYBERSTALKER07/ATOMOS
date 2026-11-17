import Foundation
import Observation

@Observable
@MainActor
final class AccountProfileViewModel {
    var retailerId = ""
    var name = ""
    var company = ""
    var phone = ""
    var receivingWindowOpen = ""
    var receivingWindowClose = ""
    var isLoading = false
    var isSaving = false
    var errorMessage: String?
    var saveMessage: String?
    var openWindowError: String?
    var closeWindowError: String?

    private let api = APIClient.shared

    func clearOpenWindowError() {
        openWindowError = nil
    }

    func clearCloseWindowError() {
        closeWindowError = nil
    }

    func loadProfile() async {
        isLoading = true
        errorMessage = nil
        defer { isLoading = false }
        do {
            let profile = try await api.getProfile()
            retailerId = profile.id
            name = profile.name
            company = profile.company
            phone = profile.phone
            receivingWindowOpen = profile.receivingWindowOpen ?? ""
            receivingWindowClose = profile.receivingWindowClose ?? ""
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func saveProfile() async {
        openWindowError = validateReceivingWindow(receivingWindowOpen)
        closeWindowError = validateReceivingWindow(receivingWindowClose)
        if openWindowError != nil || closeWindowError != nil {
            Haptics.error()
            return
        }

        isSaving = true
        errorMessage = nil
        saveMessage = nil
        defer { isSaving = false }

        do {
            _ = try await RetailerProfileService.saveProfile(
                api: api,
                retailerId: retailerId,
                name: name.trimmingCharacters(in: .whitespacesAndNewlines),
                company: company.trimmingCharacters(in: .whitespacesAndNewlines),
                phone: nil,
                location: nil,
                receivingWindowOpen: normalizeReceivingWindow(receivingWindowOpen),
                receivingWindowClose: normalizeReceivingWindow(receivingWindowClose)
            )
            saveMessage = "Profile saved"
            Haptics.success()
            await loadProfile()
        } catch {
            errorMessage = error.localizedDescription
            Haptics.error()
        }
    }

    private func validateReceivingWindow(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        let normalized = normalizeReceivingWindow(trimmed)
        guard normalized.range(of: "^([01]\\d|2[0-3]):[0-5]\\d$", options: .regularExpression) != nil else {
            return "Use 24-hour HH:MM format (e.g. 09:00)"
        }
        return nil
    }

    private func normalizeReceivingWindow(_ raw: String) -> String {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "" }
        let parts = trimmed.split(separator: ":", omittingEmptySubsequences: false)
        guard parts.count == 2,
              let hour = Int(parts[0]), (0...23).contains(hour),
              let minute = Int(parts[1]), (0...59).contains(minute) else {
            return trimmed
        }
        return String(format: "%02d:%02d", hour, minute)
    }
}
