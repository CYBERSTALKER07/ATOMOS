import Foundation
import Observation

@Observable
@MainActor
final class StaffViewModel {
    typealias FetchStaff = @MainActor () async throws -> StaffListResponse
    typealias CreateStaff = @MainActor (String, String, String) async throws -> CreateStaffResponse

    var staff: [StaffMember] = []
    var loading = true
    var error: String?
    var showCreate = false
    var createdPin: String?
    var showPinAlert = false

    var draftName = ""
    var draftPhone = ""
    var draftRole = "WAREHOUSE_ADMIN"
    var submitting = false
    var createError: String?

    var canCreate: Bool {
        !submitting
            && !draftName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !draftPhone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private let fetchStaff: FetchStaff
    private let createStaffFn: CreateStaff

    init(
        fetchStaff: FetchStaff? = nil,
        createStaff: CreateStaff? = nil
    ) {
        self.fetchStaff = fetchStaff ?? {
            try await WarehouseService.staff()
        }
        self.createStaffFn = createStaff ?? { name, phone, role in
            try await WarehouseService.createStaff(name: name, phone: phone, role: role)
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let resp = try await fetchStaff()
            staff = resp.staff
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func prepareCreateSheet() {
        draftName = ""
        draftPhone = ""
        draftRole = "WAREHOUSE_ADMIN"
        createError = nil
        submitting = false
        showCreate = true
    }

    func createStaff() async -> Bool {
        let name = draftName.trimmingCharacters(in: .whitespacesAndNewlines)
        let phone = draftPhone.trimmingCharacters(in: .whitespacesAndNewlines)
        let role = draftRole.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            ? "WAREHOUSE_ADMIN"
            : draftRole.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty, !phone.isEmpty else { return false }

        submitting = true
        createError = nil
        defer { submitting = false }

        do {
            let resp = try await createStaffFn(name, phone, role)
            showCreate = false
            createdPin = resp.pin
            showPinAlert = true
            PegasusHaptics.success()
            await load()
            return true
        } catch {
            createError = error.localizedDescription
            PegasusHaptics.error()
            return false
        }
    }

    func acknowledgePin() {
        showPinAlert = false
        createdPin = nil
    }

    func handleReconnectDuringCreate() {
        if submitting {
            submitting = false
            createError = "Connection restored — verify status before retrying."
        }
    }
}
