import Foundation
import Observation

struct PickedMigrationArtifact: Identifiable {
    let id = UUID()
    let fileName: String
    let data: Data
    let sha256: String
}

@Observable
@MainActor
final class MigrationViewModel {
    var projects: [MigrationProject] = []
    var selectedProjectId: String?
    var tree: MigrationProjectTree?
    var selectedSessionId: String?
    var warehouses: [SupplierTopologyWarehouse] = []

    var loading = false
    var busy = false
    var error: String?
    var notice: String?

    // New Project Sheet
    var showCreateProjectSheet = false
    var newProjectName = ""
    var newProjectWarehouseId = ""

    // Bundle Upload Progress
    var uploading = false
    var uploadedCount = 0
    var totalUploadCount = 0

    // Auto-Sort Review Overrides (keyed by "\(artifactId)::\(streamKey)")
    var packOverrides: [String: String] = [:]
    var encodingOverrides: [String: String] = [:]
    var ignoredStreams: [String: Bool] = [:]
    var resolvedInventoryMode = "lots"
    var activeReviewBundleTree: MigrationBundleTree?

    // Column Mapper State
    var mappingSessionId: String?
    var mappingLoading = false
    var mappingSaving = false
    var taxonomy: [MigrationCanonicalField] = []
    var mappingDoc: MigrationSessionMappingDocument?
    var proposals: [MigrationColumnMappingProposal] = []
    var sampleRows: [MigrationStagedRow] = []
    var saveMappingTemplate = true

    // Dry-Run & Cutover State
    var applyMode = "MERGE"
    var setOnHandConfirmed = false
    var dryRunResult: MigrationDryRunResponse?
    var applyResult: MigrationApplyResult?
    var rollbackResult: MigrationRollbackResult?
    var releaseResult: MigrationOpenOrderReleaseResponse?
    var cutoverResult: MigrationCutoverResult?
    var openOrderIdsText = ""
    var releaseTargetStatus = "PENDING"
    var quarantinedErrorsCSV: String?

    var selectedSession: MigrationSession? {
        guard let sid = selectedSessionId else { return tree?.sessions.first }
        return tree?.sessions.first { $0.sessionId == sid } ?? tree?.sessions.first
    }

    var sortedSessions: [MigrationSession] {
        guard let tree else { return [] }
        let orderIndex = Dictionary(uniqueKeysWithValues: tree.dag.order.enumerated().map { ($1, $0) })
        return tree.sessions.sorted { a, b in
            let ia = orderIndex[a.pack ?? ""] ?? 99
            let ib = orderIndex[b.pack ?? ""] ?? 99
            if ia != ib { return ia < ib }
            return a.fileName < b.fileName
        }
    }

    func load(silent: Bool = false) async {
        if !silent && projects.isEmpty {
            loading = true
        }
        if !silent {
            error = nil
        }
        defer { loading = false }
        do {
            async let projTask = MigrationService.listProjects()
            async let topoTask: SupplierTopologyResponse? = try? await SupplierOperationsService.topology()
            let loadedProjects = try await projTask
            let topo = await topoTask
            projects = loadedProjects
            if let topo {
                warehouses = topo.warehouses
            }
            if selectedProjectId == nil || !loadedProjects.contains(where: { $0.projectId == selectedProjectId }) {
                selectedProjectId = loadedProjects.first?.projectId
            }
            if let pid = selectedProjectId {
                try await refreshTree(projectId: pid)
            } else {
                tree = nil
                selectedSessionId = nil
            }
        } catch {
            if !silent || projects.isEmpty {
                SupplierActionFailure.record(underlying: error, hasContent: !projects.isEmpty, error: &self.error)
            }
        }
    }

    func refreshTree(projectId: String? = nil) async throws {
        guard let pid = projectId ?? selectedProjectId else { return }
        let loaded = try await MigrationService.getProjectTree(projectId: pid)
        tree = loaded
        if let sid = selectedSessionId, loaded.sessions.contains(where: { $0.sessionId == sid }) {
            // keep selection
        } else {
            selectedSessionId = loaded.sessions.first?.sessionId
        }
        syncSortDrafts(from: loaded.bundles.first?.bundle.sortSummary)
    }

    func syncSortDrafts(from summary: MigrationSortSummary?) {
        guard let streams = summary?.streams else { return }
        for st in streams {
            let key = st.id
            if packOverrides[key] == nil {
                packOverrides[key] = st.detectedPacks.first ?? ""
            }
            if encodingOverrides[key] == nil {
                encodingOverrides[key] = st.encoding.isEmpty ? "utf-8" : st.encoding
            }
            if ignoredStreams[key] == nil {
                ignoredStreams[key] = st.ignored ?? false
            }
        }
    }

    func createProject() async {
        let trimmed = newProjectName.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !busy else { return }
        busy = true
        error = nil
        notice = nil
        defer { busy = false }
        do {
            let scopeId = SupplierIdempotencyKeys.supplierScopeId()
            let created = try await MigrationService.createProject(
                name: trimmed,
                defaultWarehouseId: newProjectWarehouseId.isEmpty ? nil : newProjectWarehouseId,
                idempotencyKey: SupplierIdempotencyKeys.migrationProjectCreate(scopeId: scopeId, name: trimmed)
            )
            newProjectName = ""
            showCreateProjectSheet = false
            selectedProjectId = created.projectId
            await load()
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !projects.isEmpty, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func uploadPickedFiles(_ picked: [PickedMigrationArtifact]) async {
        guard let pid = selectedProjectId, !picked.isEmpty, !uploading else { return }
        let batch = Array(picked.prefix(50))
        uploading = true
        uploadedCount = 0
        totalUploadCount = batch.count
        error = nil
        notice = nil
        defer { uploading = false }

        do {
            let fingerprint = batch.map { "\($0.fileName):\($0.sha256)" }.joined(separator: "|")
            let declareFiles = batch.map {
                MigrationDeclareFile(
                    fileName: $0.fileName,
                    byteSize: Int64($0.data.count),
                    contentSha256: $0.sha256
                )
            }
            let created = try await MigrationService.createBundle(
                projectId: pid,
                files: declareFiles,
                idempotencyKey: SupplierIdempotencyKeys.migrationBundleCreate(projectId: pid, fingerprint: fingerprint)
            )
            var uploadedMeta: [String: MigrationUploadedArtifactMeta] = [:]
            for (idx, ticket) in created.uploadTickets.enumerated() {
                guard idx < batch.count else { continue }
                let artifact = batch[idx]
                try await MigrationService.uploadArtifactToSignedTicket(ticket: ticket, data: artifact.data)
                uploadedMeta[ticket.artifactId] = MigrationUploadedArtifactMeta(
                    byteSize: Int64(artifact.data.count),
                    contentSha256: artifact.sha256
                )
                uploadedCount = idx + 1
            }
            let completed = try await MigrationService.markBundleUploaded(
                projectId: pid,
                bundleId: created.bundle.bundleId,
                artifacts: uploadedMeta,
                idempotencyKey: SupplierIdempotencyKeys.migrationBundleUploaded(bundleId: created.bundle.bundleId, fingerprint: fingerprint),
                sync: true
            )
            if let summary = completed.summary {
                syncSortDrafts(from: summary)
            }
            try await refreshTree(projectId: pid)
            notice = L10n.format("supplier_portal.migrations.bundle_status", completed.bundle.status)
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: tree != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func saveSortOverride(bundleTree: MigrationBundleTree) async {
        guard let pid = selectedProjectId,
              let streams = bundleTree.bundle.sortSummary?.streams,
              !busy else { return }
        busy = true
        error = nil
        defer { busy = false }

        do {
            let overrides = streams.map { st in
                let k = st.id
                let chosenPack = packOverrides[k] ?? st.detectedPacks.first ?? ""
                return MigrationStreamOverrideItem(
                    artifactId: st.artifactId,
                    streamKey: st.streamKey,
                    packs: chosenPack.isEmpty ? nil : [chosenPack],
                    ignore: ignoredStreams[k] ?? false,
                    encoding: encodingOverrides[k] ?? st.encoding,
                    confirmed: true
                )
            }
            let fp = SupplierIdempotencyKeys.encodedPayload(overrides) + "|\(resolvedInventoryMode)"
            let req = MigrationBundleSortOverrideRequest(
                overrides: overrides,
                resolvedInventoryMode: bundleTree.bundle.sortSummary?.conflictingInventoryMode == true ? resolvedInventoryMode : nil,
                confirmReview: true
            )
            let res = try await MigrationService.overrideBundleSort(
                projectId: pid,
                bundleId: bundleTree.bundle.bundleId,
                request: req,
                idempotencyKey: SupplierIdempotencyKeys.migrationBundleSort(bundleId: bundleTree.bundle.bundleId, fingerprint: fp)
            )
            activeReviewBundleTree = nil
            selectedSessionId = res.sessions.first?.sessionId
            try await refreshTree(projectId: pid)
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: tree != nil, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func openMapper(for sessionId: String) async {
        selectedSessionId = sessionId
        mappingSessionId = sessionId
        mappingLoading = true
        error = nil
        defer { mappingLoading = false }
        do {
            async let mapTask = MigrationService.getSessionMapping(sessionId: sessionId)
            async let rowsTask = MigrationService.getSessionRows(sessionId: sessionId, limit: 20, offset: 0)
            let mapRes = try await mapTask
            let rowsRes = try await rowsTask
            taxonomy = mapRes.taxonomy
            mappingDoc = mapRes.mapping
            proposals = mapRes.mapping.mappings
            sampleRows = rowsRes.rows
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
        }
    }

    func samplePreview(for sourceColumn: String) -> String {
        let vals = sampleRows.prefix(5).compactMap { row -> String? in
            let v = row.rawData[sourceColumn]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            return v.isEmpty ? nil : String(v.prefix(28))
        }.prefix(3)
        return vals.isEmpty ? "—" : vals.joined(separator: " | ")
    }

    func updateProposalTarget(sourceColumn: String, targetField: String) {
        guard let idx = proposals.firstIndex(where: { $0.sourceColumn == sourceColumn }) else { return }
        proposals[idx].targetField = targetField
        proposals[idx].confirmed = !targetField.isEmpty
        if !targetField.isEmpty {
            proposals[idx].confidence = max(proposals[idx].confidence, 1.0)
            proposals[idx].source = "manual"
        }
    }

    func acceptHighConfidenceMappings() async {
        guard let sid = mappingSessionId, !mappingSaving else { return }
        mappingSaving = true
        error = nil
        defer { mappingSaving = false }
        do {
            let fp = (mappingDoc?.mappingHash ?? sid) + "-accept80"
            let res = try await MigrationService.confirmSessionMapping(
                sessionId: sid,
                request: MigrationConfirmMappingRequest(
                    mappings: proposals,
                    acceptAboveConfidence: 0.80,
                    saveToLibrary: saveMappingTemplate,
                    strict: false
                ),
                idempotencyKey: SupplierIdempotencyKeys.migrationMappingConfirm(sessionId: sid, fingerprint: fp)
            )
            mappingDoc = res.mapping
            proposals = res.mapping.mappings
            try await refreshTree()
            if res.mapping.allRequiredMapped {
                mappingSessionId = nil
            }
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func confirmMappings() async {
        guard let sid = mappingSessionId, !mappingSaving else { return }
        mappingSaving = true
        error = nil
        defer { mappingSaving = false }
        do {
            let finalProposals = proposals.map { item -> MigrationColumnMappingProposal in
                var copy = item
                copy.confirmed = !item.targetField.isEmpty
                return copy
            }
            let fp = SupplierIdempotencyKeys.encodedPayload(finalProposals)
            let res = try await MigrationService.confirmSessionMapping(
                sessionId: sid,
                request: MigrationConfirmMappingRequest(
                    mappings: finalProposals,
                    acceptAboveConfidence: nil,
                    saveToLibrary: saveMappingTemplate,
                    strict: true
                ),
                idempotencyKey: SupplierIdempotencyKeys.migrationMappingConfirm(sessionId: sid, fingerprint: fp)
            )
            mappingDoc = res.mapping
            proposals = res.mapping.mappings
            mappingSessionId = nil
            try await refreshTree()
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func runDryRun(sessionOnly: Bool) async {
        guard let pid = selectedProjectId, !busy else { return }
        busy = true
        error = nil
        quarantinedErrorsCSV = nil
        defer { busy = false }
        do {
            let res: MigrationDryRunResponse
            if sessionOnly, let sid = selectedSession?.sessionId {
                res = try await MigrationService.runSessionDryRun(
                    sessionId: sid,
                    applyMode: applyMode,
                    idempotencyKey: SupplierIdempotencyKeys.migrationDryRun(scopeId: sid, applyMode: applyMode)
                )
            } else {
                res = try await MigrationService.runProjectDryRun(
                    projectId: pid,
                    applyMode: applyMode,
                    idempotencyKey: SupplierIdempotencyKeys.migrationDryRun(scopeId: pid, applyMode: applyMode)
                )
            }
            dryRunResult = res
            try await refreshTree(projectId: pid)
            PegasusHaptics.selection()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func buildQuarantinedCSV() async {
        guard let sid = selectedSession?.sessionId else { return }
        do {
            let res = try await MigrationService.getSessionRows(sessionId: sid, limit: 500, offset: 0, invalidOnly: true)
            var lines = ["row_index,validation_errors,raw_json"]
            for r in res.rows {
                let errs = r.validationErrors.joined(separator: "; ").replacingOccurrences(of: "\"", with: "\"\"")
                let rawPairs = r.rawData.sorted(by: { $0.key < $1.key }).map { "\($0.key)=\($0.value)" }.joined(separator: " | ")
                    .replacingOccurrences(of: "\"", with: "\"\"")
                lines.append("\(r.rowIndex),\"\(errs)\",\"\(rawPairs)\"")
            }
            quarantinedErrorsCSV = lines.joined(separator: "\n")
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
        }
    }

    func approveSession() async {
        guard let sid = selectedSession?.sessionId, !busy else { return }
        busy = true
        error = nil
        defer { busy = false }
        do {
            let reportId = dryRunResult?.report.reportId ?? selectedSession?.dryRunReportId
            _ = try await MigrationService.approveSession(
                sessionId: sid,
                request: MigrationApproveRequest(
                    applyMode: applyMode,
                    setOnHandConfirmed: applyMode == "SET_ON_HAND" ? setOnHandConfirmed : nil,
                    expectedReportId: reportId
                ),
                idempotencyKey: SupplierIdempotencyKeys.migrationSessionApprove(sessionId: sid, reportId: reportId)
            )
            try await refreshTree()
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func applySession() async {
        guard let sid = selectedSession?.sessionId, !busy else { return }
        busy = true
        error = nil
        defer { busy = false }
        do {
            let res = try await MigrationService.applySession(
                sessionId: sid,
                applyMode: applyMode,
                chunkSize: 500,
                idempotencyKey: SupplierIdempotencyKeys.migrationSessionApply(
                    sessionId: sid,
                    fingerprint: selectedSession?.dryRunFingerprint
                )
            )
            applyResult = res
            try await refreshTree()
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func rollbackSession() async {
        guard let sid = selectedSession?.sessionId, !busy else { return }
        busy = true
        error = nil
        defer { busy = false }
        do {
            let res = try await MigrationService.rollbackSession(
                sessionId: sid,
                idempotencyKey: SupplierIdempotencyKeys.migrationSessionRollback(sessionId: sid)
            )
            rollbackResult = res
            try await refreshTree()
            PegasusHaptics.warning()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func releaseOpenOrders() async {
        guard let pid = selectedProjectId, !busy else { return }
        let ids = openOrderIdsText
            .components(separatedBy: CharacterSet(charactersIn: ", \n\t"))
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        guard !ids.isEmpty else { return }
        busy = true
        error = nil
        defer { busy = false }
        do {
            let res = try await MigrationService.releaseOpenOrders(
                projectId: pid,
                orderIds: ids,
                targetStatus: releaseTargetStatus,
                idempotencyKey: SupplierIdempotencyKeys.migrationOpenOrdersRelease(
                    scopeId: pid,
                    orderIds: ids,
                    targetStatus: releaseTargetStatus
                )
            )
            releaseResult = res
            try await refreshTree(projectId: pid)
            PegasusHaptics.confirm()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }

    func cutoverProject() async {
        guard let pid = selectedProjectId, !busy else { return }
        busy = true
        error = nil
        defer { busy = false }
        do {
            let res = try await MigrationService.cutoverProject(
                projectId: pid,
                idempotencyKey: SupplierIdempotencyKeys.migrationCutover(projectId: pid)
            )
            cutoverResult = res
            try await refreshTree(projectId: pid)
            PegasusHaptics.success()
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: true, error: &self.error)
            PegasusHaptics.error()
        }
    }
}
