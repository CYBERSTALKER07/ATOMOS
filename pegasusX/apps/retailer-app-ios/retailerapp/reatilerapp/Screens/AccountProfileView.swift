import SwiftUI

struct AccountProfileView: View {
    @State private var retailerId = ""
    @State private var name = ""
    @State private var company = ""
    @State private var phone = ""
    @State private var receivingWindowOpen = ""
    @State private var receivingWindowClose = ""
    @State private var isLoading = false
    @State private var isSaving = false
    @State private var errorMessage: String?
    @State private var saveMessage: String?
    @State private var openWindowError: String?
    @State private var closeWindowError: String?

    private let api = APIClient.shared

    var body: some View {
        Form {
            if let errorMessage {
                Section {
                    Text(errorMessage)
                        .font(.footnote)
                        .foregroundStyle(AppTheme.destructive)
                }
            }
            if let saveMessage {
                Section {
                    Text(saveMessage)
                        .font(.footnote)
                        .foregroundStyle(AppTheme.success)
                }
            }

            Section {
                TextField("Entity name", text: $name)
                TextField("Company", text: $company)
                TextField("Phone", text: $phone)
                    .disabled(true)
            } header: {
                Text("Business")
            } footer: {
                Text("Receiving hours feed dispatch SLA scheduling for your deliveries.")
            }

            Section("Receiving window") {
                TextField("Opens (HH:MM)", text: $receivingWindowOpen)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.numbersAndPunctuation)
                    .onChange(of: receivingWindowOpen) { _, _ in openWindowError = nil }

                if let openWindowError {
                    Text(openWindowError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.error)
                }

                TextField("Closes (HH:MM)", text: $receivingWindowClose)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.numbersAndPunctuation)
                    .onChange(of: receivingWindowClose) { _, _ in closeWindowError = nil }

                if let closeWindowError {
                    Text(closeWindowError)
                        .font(.caption)
                        .foregroundStyle(AppTheme.error)
                }
            }

            Section {
                Button {
                    Task { await saveProfile() }
                } label: {
                    HStack {
                        Spacer()
                        if isSaving {
                            ProgressView()
                        } else {
                            Text("Save profile")
                                .fontWeight(.semibold)
                        }
                        Spacer()
                    }
                }
                .disabled(isLoading || isSaving)
            }
        }
        .navigationTitle("Account")
        .navigationBarTitleDisplayMode(.inline)
        .task { await loadProfile() }
        .refreshable { await loadProfile() }
        .overlay {
            if isLoading && name.isEmpty && company.isEmpty {
                ProgressView()
            }
        }
    }

    private func loadProfile() async {
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

    private func saveProfile() async {
        openWindowError = validateReceivingWindow(receivingWindowOpen)
        closeWindowError = validateReceivingWindow(receivingWindowClose)
        if openWindowError != nil || closeWindowError != nil {
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
                receivingWindowClose: normalizeReceivingWindow(receivingWindowClose),
            )
            saveMessage = "Profile saved"
            await loadProfile()
        } catch {
            errorMessage = error.localizedDescription
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
