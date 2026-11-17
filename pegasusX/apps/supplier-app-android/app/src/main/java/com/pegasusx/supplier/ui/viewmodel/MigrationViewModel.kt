package com.pegasusx.supplier.ui.viewmodel

import android.content.Context
import android.net.Uri
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.model.*
import com.pegasusx.supplier.data.remote.MigrationRepository
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.*
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import javax.inject.Inject

data class MigrationUiState(
    val projects: List<MigrationProject> = emptyList(),
    val selectedProjectId: String? = null,
    val tree: MigrationProjectTree? = null,
    val selectedSessionId: String? = null,
    val warehouses: List<SupplierTopologyWarehouse> = emptyList(),
    val loading: Boolean = true,
    val busy: Boolean = false,
    val error: String? = null,
    val notice: String? = null,
    // Create project dialog
    val showCreateProjectDialog: Boolean = false,
    val newProjectName: String = "",
    val newProjectWarehouseId: String = "",
    // Upload progress
    val uploading: Boolean = false,
    val uploadedCount: Int = 0,
    val totalUploadCount: Int = 0,
    // Auto-Sort Review
    val activeReviewBundle: MigrationBundleTree? = null,
    val packOverrides: Map<String, String> = emptyMap(),
    val encodingOverrides: Map<String, String> = emptyMap(),
    val ignoredStreams: Map<String, Boolean> = emptyMap(),
    val resolvedInventoryMode: String = "lots",
    // Column Mapper
    val mappingSessionId: String? = null,
    val mappingLoading: Boolean = false,
    val mappingSaving: Boolean = false,
    val taxonomy: List<MigrationCanonicalField> = emptyList(),
    val mappingDoc: MigrationSessionMappingDocument? = null,
    val proposals: List<MigrationColumnMappingProposal> = emptyList(),
    val sampleRows: List<MigrationStagedRow> = emptyList(),
    val saveMappingTemplate: Boolean = true,
    // Dry-Run & Cutover
    val applyMode: String = "MERGE",
    val setOnHandConfirmed: Boolean = false,
    val dryRunResult: MigrationDryRunResponse? = null,
    val applyResult: MigrationApplyResult? = null,
    val rollbackResult: MigrationRollbackResult? = null,
    val releaseResult: MigrationOpenOrderReleaseResponse? = null,
    val cutoverResult: MigrationCutoverResult? = null,
    val openOrderIdsText: String = "",
    val releaseTargetStatus: String = "PENDING",
    val quarantinedErrorsCSV: String? = null,
) {
    val selectedSession: MigrationSession?
        get() = selectedSessionId?.let { id -> tree?.sessions?.firstOrNull { it.sessionId == id } }
            ?: tree?.sessions?.firstOrNull()

    val sortedSessions: List<MigrationSession>
        get() {
            val t = tree ?: return emptyList()
            val orderIndex = t.dag.order.withIndex().associate { it.value to it.index }
            return t.sessions.sortedWith(
                compareBy<MigrationSession> { orderIndex[it.pack.orEmpty()] ?: 99 }
                    .thenBy { it.fileName },
            )
        }
}

@HiltViewModel
class MigrationViewModel @Inject constructor(
    private val repo: MigrationRepository,
    private val ops: SupplierOperationsRepository,
    private val realtimeSignals: SupplierRealtimeSignals,
    private val json: Json,
) : ViewModel() {
    private val _state = MutableStateFlow(MigrationUiState())
    val state: StateFlow<MigrationUiState> = _state.asStateFlow()

    val migrationProgressBySession: StateFlow<Map<String, MigrationApplyProgressEvent>> =
        realtimeSignals.migrationProgressBySession

    init {
        load()
        viewModelScope.launch {
            realtimeSignals.refreshTick.collect { load(silent = true) }
        }
        viewModelScope.launch {
            realtimeSignals.reconnectTick.collect { load(silent = true) }
        }
    }

    fun update(block: MigrationUiState.() -> MigrationUiState) {
        _state.update(block)
    }

    fun clearBanner() {
        _state.update { it.copy(error = null, notice = null) }
    }

    fun selectProject(projectId: String) {
        _state.update { it.copy(selectedProjectId = projectId) }
        viewModelScope.launch { refreshTree(projectId) }
    }

    fun load(silent: Boolean = false) {
        viewModelScope.launch {
            if (!silent && _state.value.projects.isEmpty()) {
                _state.update { it.copy(loading = true, error = null) }
            } else if (!silent) {
                _state.update { it.copy(error = null) }
            }
            try {
                val projResp = repo.listProjects()
                val topoResp = runCatching { ops.getTopology() }.getOrNull()
                if (!projResp.isSuccessful) {
                    if (!silent || _state.value.projects.isEmpty()) {
                        _state.update { it.copy(loading = false, error = "Failed to load projects (${projResp.code()})") }
                    } else {
                        _state.update { it.copy(loading = false) }
                    }
                    return@launch
                }
                val loadedProjects = projResp.body()?.projects.orEmpty()
                val currentPid = _state.value.selectedProjectId
                val nextPid = if (currentPid != null && loadedProjects.any { it.projectId == currentPid }) {
                    currentPid
                } else {
                    loadedProjects.firstOrNull()?.projectId
                }
                val loadedWarehouses = topoResp?.takeIf { it.isSuccessful }?.body()?.warehouses
                    ?: _state.value.warehouses
                _state.update {
                    it.copy(
                        projects = loadedProjects,
                        selectedProjectId = nextPid,
                        warehouses = loadedWarehouses,
                        loading = false,
                    )
                }
                if (nextPid != null) {
                    refreshTree(nextPid)
                } else {
                    _state.update { it.copy(tree = null, selectedSessionId = null) }
                }
            } catch (e: Exception) {
                if (!silent || _state.value.projects.isEmpty()) {
                    _state.update { it.copy(loading = false, error = e.message ?: "Network error") }
                } else {
                    _state.update { it.copy(loading = false) }
                }
            }
        }
    }

    private suspend fun refreshTree(projectId: String) {
        val resp = repo.getProjectTree(projectId)
        if (!resp.isSuccessful) {
            _state.update { it.copy(error = "Failed to load project tree (${resp.code()})") }
            return
        }
        val tree = resp.body() ?: return
        val currentSid = _state.value.selectedSessionId
        val nextSid = if (currentSid != null && tree.sessions.any { it.sessionId == currentSid }) {
            currentSid
        } else {
            tree.sessions.firstOrNull()?.sessionId
        }
        _state.update { st ->
            st.copy(
                tree = tree,
                selectedSessionId = nextSid,
            )
        }
        syncSortDrafts(tree.bundles.firstOrNull()?.bundle?.sortSummary)
    }

    fun syncSortDrafts(summary: MigrationSortSummary?) {
        val streams = summary?.streams ?: return
        _state.update { st ->
            val packs = st.packOverrides.toMutableMap()
            val encs = st.encodingOverrides.toMutableMap()
            val ignored = st.ignoredStreams.toMutableMap()
            for (stream in streams) {
                val key = stream.id
                packs.putIfAbsent(key, stream.detectedPacks.firstOrNull().orEmpty())
                encs.putIfAbsent(key, stream.encoding.ifBlank { "utf-8" })
                ignored.putIfAbsent(key, stream.ignored)
            }
            st.copy(
                packOverrides = packs,
                encodingOverrides = encs,
                ignoredStreams = ignored,
            )
        }
    }

    fun createProject() {
        val name = _state.value.newProjectName.trim()
        if (name.isEmpty() || _state.value.busy) return
        val whId = _state.value.newProjectWarehouseId.ifBlank { null }
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null, notice = null) }
            try {
                val scopeId = SupplierIdempotencyKeys.supplierScopeId()
                val resp = repo.createProject(
                    name = name,
                    defaultWarehouseId = whId,
                    idempotencyKey = SupplierIdempotencyKeys.migrationProjectCreate(scopeId, name),
                )
                val created = resp.body()
                if (resp.isSuccessful && created != null) {
                    _state.update {
                        it.copy(
                            busy = false,
                            newProjectName = "",
                            showCreateProjectDialog = false,
                            selectedProjectId = created.projectId,
                        )
                    }
                    load()
                } else {
                    _state.update { it.copy(busy = false, error = "Create project failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Create failed") }
            }
        }
    }

    fun uploadPickedUris(context: Context, uris: List<Uri>) {
        val pid = _state.value.selectedProjectId ?: return
        if (uris.isEmpty() || _state.value.uploading) return
        viewModelScope.launch {
            _state.update {
                it.copy(
                    uploading = true,
                    uploadedCount = 0,
                    totalUploadCount = uris.size.coerceAtMost(50),
                    error = null,
                    notice = null,
                )
            }
            try {
                val files = repo.readPickedFiles(context, uris)
                if (files.isEmpty()) {
                    _state.update { it.copy(uploading = false, error = "No readable files selected") }
                    return@launch
                }
                _state.update { it.copy(totalUploadCount = files.size) }
                val fingerprint = files.joinToString("|") { "${it.fileName}:${it.sha256}" }
                val declared = files.map { MigrationDeclareFile(it.fileName, it.bytes.size.toLong(), it.sha256) }
                val bundleResp = repo.createBundle(
                    projectId = pid,
                    files = declared,
                    idempotencyKey = SupplierIdempotencyKeys.migrationBundleCreate(pid, fingerprint),
                )
                val created = bundleResp.body()
                if (!bundleResp.isSuccessful || created == null) {
                    _state.update { it.copy(uploading = false, error = "Bundle creation failed (${bundleResp.code()})") }
                    return@launch
                }

                val uploadedMeta = mutableMapOf<String, MigrationUploadedArtifactMeta>()
                created.uploadTickets.forEachIndexed { idx, ticket ->
                    if (idx < files.size) {
                        val file = files[idx]
                        repo.uploadArtifactToSignedTicket(ticket, file.bytes)
                        uploadedMeta[ticket.artifactId] = MigrationUploadedArtifactMeta(
                            byteSize = file.bytes.size.toLong(),
                            contentSha256 = file.sha256,
                        )
                        _state.update { it.copy(uploadedCount = idx + 1) }
                    }
                }

                val completeResp = repo.markBundleUploaded(
                    projectId = pid,
                    bundleId = created.bundle.bundleId,
                    artifacts = uploadedMeta,
                    idempotencyKey = SupplierIdempotencyKeys.migrationBundleUploaded(created.bundle.bundleId, fingerprint),
                )
                if (completeResp.isSuccessful) {
                    val completed = completeResp.body()
                    syncSortDrafts(completed?.summary)
                    refreshTree(pid)
                    _state.update {
                        it.copy(
                            uploading = false,
                            notice = "Bundle uploaded (${completed?.bundle?.status.orEmpty()})",
                        )
                    }
                } else {
                    _state.update { it.copy(uploading = false, error = "Finalize bundle failed (${completeResp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(uploading = false, error = e.message ?: "Upload failed") }
            }
        }
    }

    fun saveSortOverride(bundleTree: MigrationBundleTree) {
        val st = _state.value
        val pid = st.selectedProjectId ?: return
        val streams = bundleTree.bundle.sortSummary?.streams ?: return
        if (st.busy) return
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null) }
            try {
                val overrides = streams.map { stream ->
                    val key = stream.id
                    val chosenPack = st.packOverrides[key] ?: stream.detectedPacks.firstOrNull().orEmpty()
                    MigrationStreamOverrideItem(
                        artifactId = stream.artifactId,
                        streamKey = stream.streamKey,
                        packs = if (chosenPack.isBlank()) null else listOf(chosenPack),
                        ignore = st.ignoredStreams[key] ?: false,
                        encoding = st.encodingOverrides[key] ?: stream.encoding,
                        confirmed = true,
                    )
                }
                val fp = json.encodeToString(overrides) + "|${st.resolvedInventoryMode}"
                val req = MigrationBundleSortOverrideRequest(
                    overrides = overrides,
                    resolvedInventoryMode = if (bundleTree.bundle.sortSummary?.conflictingInventoryMode == true) {
                        st.resolvedInventoryMode
                    } else {
                        null
                    },
                    confirmReview = true,
                )
                val resp = repo.overrideBundleSort(
                    projectId = pid,
                    bundleId = bundleTree.bundle.bundleId,
                    request = req,
                    idempotencyKey = SupplierIdempotencyKeys.migrationBundleSort(bundleTree.bundle.bundleId, fp),
                )
                if (resp.isSuccessful) {
                    val body = resp.body()
                    _state.update {
                        it.copy(
                            busy = false,
                            activeReviewBundle = null,
                            selectedSessionId = body?.sessions?.firstOrNull()?.sessionId ?: it.selectedSessionId,
                        )
                    }
                    refreshTree(pid)
                } else {
                    _state.update { it.copy(busy = false, error = "Sort override failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Sort failed") }
            }
        }
    }

    fun openMapper(sessionId: String) {
        viewModelScope.launch {
            _state.update {
                it.copy(
                    selectedSessionId = sessionId,
                    mappingSessionId = sessionId,
                    mappingLoading = true,
                    error = null,
                )
            }
            try {
                val mapResp = repo.getSessionMapping(sessionId)
                val rowsResp = repo.getSessionRows(sessionId, limit = 20, offset = 0)
                if (mapResp.isSuccessful) {
                    val body = mapResp.body()
                    if (body != null) {
                        _state.update {
                            it.copy(
                                mappingLoading = false,
                                taxonomy = body.taxonomy,
                                mappingDoc = body.mapping,
                                proposals = body.mapping.mappings,
                                sampleRows = rowsResp.body()?.rows.orEmpty(),
                            )
                        }
                    } else {
                        _state.update { it.copy(mappingLoading = false, error = "Empty mapping response (${mapResp.code()})") }
                    }
                } else {
                    _state.update { it.copy(mappingLoading = false, error = "Load mapping failed (${mapResp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(mappingLoading = false, error = e.message ?: "Load mapping failed") }
            }
        }
    }

    fun closeMapper() {
        _state.update { it.copy(mappingSessionId = null) }
    }

    fun samplePreview(sourceColumn: String): String {
        val values = _state.value.sampleRows
            .take(5)
            .mapNotNull { row -> row.textValue(sourceColumn).trim().takeIf { it.isNotEmpty() }?.take(28) }
            .take(3)
        return if (values.isEmpty()) "—" else values.joinToString(" | ")
    }

    fun updateProposalTarget(sourceColumn: String, targetField: String) {
        _state.update { st ->
            val updated = st.proposals.map { prop ->
                if (prop.sourceColumn == sourceColumn) {
                    prop.copy(
                        targetField = targetField,
                        confirmed = targetField.isNotBlank(),
                        confidence = if (targetField.isNotBlank()) maxOf(prop.confidence, 1.0) else prop.confidence,
                        source = if (targetField.isNotBlank()) "manual" else prop.source,
                    )
                } else {
                    prop
                }
            }
            st.copy(proposals = updated)
        }
    }

    fun acceptHighConfidenceMappings() {
        val st = _state.value
        val sid = st.mappingSessionId ?: return
        if (st.mappingSaving) return
        viewModelScope.launch {
            _state.update { it.copy(mappingSaving = true, error = null) }
            try {
                val fp = (st.mappingDoc?.mappingHash?.ifBlank { sid } ?: sid) + "-accept80"
                val resp = repo.confirmSessionMapping(
                    sessionId = sid,
                    request = MigrationConfirmMappingRequest(
                        mappings = st.proposals,
                        acceptAboveConfidence = 0.80,
                        saveToLibrary = st.saveMappingTemplate,
                        strict = false,
                    ),
                    idempotencyKey = SupplierIdempotencyKeys.migrationMappingConfirm(sid, fp),
                )
                val body = resp.body()
                if (resp.isSuccessful && body != null) {
                    _state.update {
                        it.copy(
                            mappingSaving = false,
                            mappingDoc = body.mapping,
                            proposals = body.mapping.mappings,
                            mappingSessionId = if (body.mapping.allRequiredMapped) null else it.mappingSessionId,
                        )
                    }
                    st.selectedProjectId?.let { refreshTree(it) }
                } else {
                    _state.update { it.copy(mappingSaving = false, error = "Accept >=80% failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(mappingSaving = false, error = e.message ?: "Accept failed") }
            }
        }
    }

    fun confirmMappings() {
        val st = _state.value
        val sid = st.mappingSessionId ?: return
        if (st.mappingSaving) return
        viewModelScope.launch {
            _state.update { it.copy(mappingSaving = true, error = null) }
            try {
                val normalized = st.proposals.map { it.copy(confirmed = it.targetField.isNotBlank()) }
                val fp = json.encodeToString(normalized)
                val resp = repo.confirmSessionMapping(
                    sessionId = sid,
                    request = MigrationConfirmMappingRequest(
                        mappings = normalized,
                        acceptAboveConfidence = null,
                        saveToLibrary = st.saveMappingTemplate,
                        strict = true,
                    ),
                    idempotencyKey = SupplierIdempotencyKeys.migrationMappingConfirm(sid, fp),
                )
                val body = resp.body()
                if (resp.isSuccessful && body != null) {
                    _state.update {
                        it.copy(
                            mappingSaving = false,
                            mappingDoc = body.mapping,
                            proposals = body.mapping.mappings,
                            mappingSessionId = null,
                        )
                    }
                    st.selectedProjectId?.let { refreshTree(it) }
                } else {
                    _state.update { it.copy(mappingSaving = false, error = "Confirm mapping failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(mappingSaving = false, error = e.message ?: "Confirm failed") }
            }
        }
    }

    fun runDryRun(sessionOnly: Boolean) {
        val st = _state.value
        val pid = st.selectedProjectId ?: return
        if (st.busy) return
        val sid = if (sessionOnly) st.selectedSession?.sessionId else null
        val scopeId = sid ?: pid
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null, quarantinedErrorsCSV = null) }
            try {
                val resp = repo.runDryRun(
                    projectId = pid,
                    sessionId = sid,
                    applyMode = st.applyMode,
                    idempotencyKey = SupplierIdempotencyKeys.migrationDryRun(scopeId, st.applyMode),
                )
                if (resp.isSuccessful && resp.body() != null) {
                    _state.update { it.copy(busy = false, dryRunResult = resp.body()) }
                    refreshTree(pid)
                } else {
                    _state.update { it.copy(busy = false, error = "Dry-run failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Dry-run failed") }
            }
        }
    }

    fun buildQuarantinedCSV() {
        val sid = _state.value.selectedSession?.sessionId ?: return
        viewModelScope.launch {
            try {
                val resp = repo.getSessionRows(sessionId = sid, limit = 500, offset = 0, invalidOnly = true)
                val body = resp.body()
                if (resp.isSuccessful && body != null) {
                    val lines = mutableListOf("row_index,validation_errors,raw_json")
                    for (row in body.rows) {
                        val errs = row.validationErrors.joinToString("; ").replace("\"", "\"\"")
                        val rawPairs = row.rawData.entries
                            .sortedBy { it.key }
                            .joinToString(" | ") { "${it.key}=${row.textValue(it.key)}" }
                            .replace("\"", "\"\"")
                        lines.add("${row.rowIndex},\"$errs\",\"$rawPairs\"")
                    }
                    _state.update { it.copy(quarantinedErrorsCSV = lines.joinToString("\n")) }
                }
            } catch (e: Exception) {
                _state.update { it.copy(error = e.message) }
            }
        }
    }

    fun approveSession() {
        val st = _state.value
        val session = st.selectedSession ?: return
        if (st.busy) return
        val reportId = st.dryRunResult?.report?.reportId ?: session.dryRunReportId
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null) }
            try {
                val resp = repo.approveSession(
                    sessionId = session.sessionId,
                    request = MigrationApproveRequest(
                        applyMode = st.applyMode,
                        setOnHandConfirmed = if (st.applyMode == "SET_ON_HAND") st.setOnHandConfirmed else null,
                        expectedReportId = reportId,
                    ),
                    idempotencyKey = SupplierIdempotencyKeys.migrationSessionApprove(session.sessionId, reportId),
                )
                if (resp.isSuccessful) {
                    _state.update { it.copy(busy = false) }
                    st.selectedProjectId?.let { refreshTree(it) }
                } else {
                    _state.update { it.copy(busy = false, error = "Approve failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Approve failed") }
            }
        }
    }

    fun applySession() {
        val st = _state.value
        val session = st.selectedSession ?: return
        if (st.busy) return
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null) }
            try {
                val resp = repo.applySession(
                    sessionId = session.sessionId,
                    applyMode = st.applyMode,
                    chunkSize = 500,
                    idempotencyKey = SupplierIdempotencyKeys.migrationSessionApply(
                        session.sessionId,
                        session.dryRunFingerprint,
                    ),
                )
                if (resp.isSuccessful) {
                    _state.update { it.copy(busy = false, applyResult = resp.body()) }
                    st.selectedProjectId?.let { refreshTree(it) }
                } else {
                    _state.update { it.copy(busy = false, error = "Apply failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Apply failed") }
            }
        }
    }

    fun rollbackSession() {
        val st = _state.value
        val session = st.selectedSession ?: return
        if (st.busy) return
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null) }
            try {
                val resp = repo.rollbackSession(
                    sessionId = session.sessionId,
                    idempotencyKey = SupplierIdempotencyKeys.migrationSessionRollback(session.sessionId),
                )
                if (resp.isSuccessful) {
                    _state.update { it.copy(busy = false, rollbackResult = resp.body()) }
                    st.selectedProjectId?.let { refreshTree(it) }
                } else {
                    _state.update { it.copy(busy = false, error = "Rollback failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Rollback failed") }
            }
        }
    }

    fun releaseOpenOrders() {
        val st = _state.value
        val pid = st.selectedProjectId ?: return
        val ids = st.openOrderIdsText
            .split(',', ' ', '\n', '\t')
            .map { it.trim() }
            .filter { it.isNotEmpty() }
        if (ids.isEmpty() || st.busy) return
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null) }
            try {
                val resp = repo.releaseOpenOrders(
                    projectId = pid,
                    orderIds = ids,
                    targetStatus = st.releaseTargetStatus,
                    idempotencyKey = SupplierIdempotencyKeys.migrationOpenOrdersRelease(pid, ids, st.releaseTargetStatus),
                )
                if (resp.isSuccessful) {
                    _state.update { it.copy(busy = false, releaseResult = resp.body()) }
                    refreshTree(pid)
                } else {
                    _state.update { it.copy(busy = false, error = "Release orders failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Release failed") }
            }
        }
    }

    fun cutoverProject() {
        val st = _state.value
        val pid = st.selectedProjectId ?: return
        if (st.busy) return
        viewModelScope.launch {
            _state.update { it.copy(busy = true, error = null) }
            try {
                val resp = repo.cutoverProject(
                    projectId = pid,
                    idempotencyKey = SupplierIdempotencyKeys.migrationCutover(pid),
                )
                if (resp.isSuccessful) {
                    _state.update { it.copy(busy = false, cutoverResult = resp.body()) }
                    refreshTree(pid)
                } else {
                    _state.update { it.copy(busy = false, error = "Cutover failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _state.update { it.copy(busy = false, error = e.message ?: "Cutover failed") }
            }
        }
    }
}
