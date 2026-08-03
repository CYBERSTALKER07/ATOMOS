import SwiftUI

struct OrderOpsActions: View {
    let state: String
    @Binding var proposeDate: Date
    @Binding var reasonInput: String
    @Binding var showProposeSheet: Bool
    @Binding var pendingAction: OrderMutationAction?
    let mutating: Bool
    let onLoadRecommendations: () -> Void

    var body: some View {
        if showOps(for: state) {
            Section("Quick actions") {
                if canProposeDate(state) {
                    DatePicker("Proposed delivery date", selection: $proposeDate, displayedComponents: .date)
                    Button("Propose new date") { showProposeSheet = true }
                        .disabled(mutating || reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                TextField("Reason (required for propose and cancel)", text: $reasonInput, axis: .vertical)
                    .lineLimit(2...4)
                if canOverflow(state) {
                    Button("Return to dispatch pool") { pendingAction = .overflow }
                        .disabled(mutating)
                }
                if canReject(state) {
                    Button("Cancel order", role: .destructive) { pendingAction = .reject }
                        .disabled(mutating)
                }
                if canReassign(state) {
                    Button("Reassign order") { onLoadRecommendations() }
                        .disabled(mutating)
                }
            }
        }
    }

    private func showOps(for state: String) -> Bool {
        canProposeDate(state) || canReject(state) || canOverflow(state) || canReassign(state)
    }

    private func canProposeDate(_ state: String) -> Bool {
        let s = state.uppercased()
        let terminal = s == "COMPLETED" || s == "CANCELLED"
        let inFlight = s == "LOADED" || s == "IN_TRANSIT"
        return !terminal && !inFlight
    }

    private func canReject(_ state: String) -> Bool {
        let s = state.uppercased()
        return ["PENDING", "LOADED", "IN_TRANSIT", "SCHEDULED", "AUTO_ACCEPTED", "DELAYED", "ARRIVED"].contains(s)
    }

    private func canOverflow(_ state: String) -> Bool {
        ["LOADED", "IN_TRANSIT"].contains(state.uppercased())
    }

    private func canReassign(_ state: String) -> Bool {
        ["PENDING", "LOADED", "IN_TRANSIT", "SCHEDULED", "AUTO_ACCEPTED", "DELAYED", "ARRIVED"].contains(state.uppercased())
    }
}
