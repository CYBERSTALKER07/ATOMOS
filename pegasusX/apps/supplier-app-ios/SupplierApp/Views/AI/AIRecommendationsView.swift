import SwiftUI

struct AIRecommendationsView: View {
    private let statusFilters = ["PENDING", "ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "ALL"]
    private let decisions = ["ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "REOPENED"]

    @State private var filter = "PENDING"
    @State private var loading = true
    @State private var error: String?
    @State private var items: [SupplierAIRecommendation] = []
    @State private var pendingDecisionId: String?
    @State private var feedback: String?

    var body: some View {
        Group {
            if loading && items.isEmpty {
                SupplierLoadingView(title: "Loading recommendations…")
            } else if let error, items.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                ResponsiveGridContentWrapper {
                    Section {
                        Picker("Status", selection: $filter) {
                            ForEach(statusFilters, id: \.self) { Text($0).tag($0) }
                        }
                        .pickerStyle(.segmented)
                        if let feedback {
                            Text(feedback).font(.footnote).foregroundStyle(.secondary)
                        }
                    }

                    if items.isEmpty {
                        Section {
                            Text(L10n.format("mobile_supplier.ui.no_lowercased_advisory_rows_for_this_supplier", "\(filter.lowercased())"))
                                .foregroundStyle(.secondary)
                        }
                    }

                    ForEach(items) { rec in
                        Section {
                            recommendationCard(rec)
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("mobile_supplier.ui.ai_recommendations")
        .task(id: filter) { await load() }
    }

    @ViewBuilder
    private func recommendationCard(_ rec: SupplierAIRecommendation) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(rec.action).font(.headline)
                Spacer()
                Text(rec.status).font(.caption).foregroundStyle(.secondary)
            }
            Text(String(format: "Confidence %.0f%% · Score %.2f", rec.confidence * 100, rec.score))
                .font(.caption)
                .foregroundStyle(.secondary)
            if !rec.explanation.isEmpty {
                Text(rec.explanation).font(.subheadline)
            }
            Text(L10n.format("mobile_supplier.ui.aggregatetype_aggregateid_source_source", "\(rec.aggregateType)", "\(rec.aggregateId)", "\(rec.source)"))
                .font(.caption2)
                .foregroundStyle(.secondary)

            if !rec.reasonCodes.isEmpty {
                Text(L10n.format("mobile_supplier.ui.reasons_joined", "\(rec.reasonCodes.joined(separator: ", "))"))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            ForEach(rec.evidence) { ev in
                HStack {
                    Text(ev.label).foregroundStyle(.secondary)
                    Spacer()
                    Text(ev.value)
                }
                .font(.caption)
            }

            if let decision = rec.decision, !decision.isEmpty {
                Text("Decision: \(decision) by \(rec.decidedBy ?? "operator")")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if rec.decision == nil || rec.decision?.isEmpty == true {
                HStack {
                    ForEach(decisions, id: \.self) { decision in
                        Button(decision.capitalized) {
                            Task { await record(rec, decision: decision) }
                        }
                        .font(.caption)
                        .buttonStyle(.bordered)
                        .disabled(pendingDecisionId != nil)
                    }
                }
            }
        }
    }

    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.aiRecommendations(status: filter, limit: 50)
            items = resp.items
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func record(_ rec: SupplierAIRecommendation, decision: String) async {
        pendingDecisionId = rec.recommendationId
        feedback = nil
        defer { pendingDecisionId = nil }
        do {
            let request = SupplierAIRecommendationDecisionRequest(
                recommendationId: rec.recommendationId,
                decision: decision,
                note: nil
            )
            let key = "ai-rec-\(rec.recommendationId)-\(decision)-\(UUID().uuidString)"
            let resp = try await SupplierOperationsService.recordAiRecommendationDecision(request, idempotencyKey: key)
            if let idx = items.firstIndex(where: { $0.recommendationId == resp.recommendation.recommendationId }) {
                items[idx] = resp.recommendation
            }
            feedback = "\(decision.lowercased()) recorded for \(rec.aggregateType) \(rec.aggregateId)."
        } catch {
            feedback = "Decision failed: \(error.localizedDescription)"
        }
    }
}
