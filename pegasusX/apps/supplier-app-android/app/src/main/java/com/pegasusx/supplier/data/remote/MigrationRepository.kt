package com.pegasusx.supplier.data.remote

import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import com.pegasusx.supplier.data.model.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import retrofit2.Response
import java.security.MessageDigest
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Singleton

data class PickedMigrationFile(
    val fileName: String,
    val bytes: ByteArray,
    val sha256: String,
)

@Singleton
class MigrationRepository @Inject constructor(
    private val api: SupplierApi,
    private val json: Json,
) {
    /**
     * Separate bare OkHttpClient without AuthInterceptor or CellPinInterceptor so
     * GCS signed upload URLs never receive Pegasus Authorization headers.
     */
    private var signedUploadClient: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(120, TimeUnit.SECONDS)
        .writeTimeout(120, TimeUnit.SECONDS)
        .build()

    internal fun setSignedUploadClientForTest(client: OkHttpClient) {
        signedUploadClient = client
    }

    companion object {
        const val RESUMABLE_CHUNK_SIZE = 8 * 1024 * 1024 // 8 MiB aligned to GCS 256 KiB multiple

        fun sha256Hex(bytes: ByteArray): String {
            val digest = MessageDigest.getInstance("SHA-256").digest(bytes)
            return digest.joinToString("") { "%02x".format(it) }
        }
    }

    suspend fun readPickedFiles(context: Context, uris: List<Uri>): List<PickedMigrationFile> =
        withContext(Dispatchers.IO) {
            val cr = context.contentResolver
            uris.take(50).mapNotNull { uri ->
                val fileName = resolveDisplayName(context, uri)
                val bytes = cr.openInputStream(uri)?.use { it.readBytes() } ?: return@mapNotNull null
                PickedMigrationFile(
                    fileName = fileName,
                    bytes = bytes,
                    sha256 = sha256Hex(bytes),
                )
            }
        }

    private fun resolveDisplayName(context: Context, uri: Uri): String {
        context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { cursor ->
            if (cursor.moveToFirst()) {
                val idx = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                if (idx >= 0) {
                    val name = cursor.getString(idx)
                    if (!name.isNullOrBlank()) return name
                }
            }
        }
        return uri.lastPathSegment?.substringAfterLast('/')?.ifBlank { "upload.bin" } ?: "upload.bin"
    }

    suspend fun uploadArtifactToSignedTicket(
        ticket: MigrationUploadTicket,
        bytes: ByteArray,
        onProgress: ((Long, Long) -> Unit)? = null,
    ) = withContext(Dispatchers.IO) {
        val total = bytes.size.toLong()
        if (ticket.uploadUrl.startsWith("https://local.invalid/")) {
            onProgress?.invoke(total, total)
            return@withContext
        }

        val mediaType = (ticket.contentType.ifBlank { "application/octet-stream" }).toMediaType()
        if (!ticket.resumable || bytes.size <= RESUMABLE_CHUNK_SIZE) {
            val method = when {
                ticket.method.equals("POST", ignoreCase = true) -> "PUT"
                ticket.method.isBlank() -> "PUT"
                else -> ticket.method.uppercase()
            }
            val builder = Request.Builder().url(ticket.uploadUrl)
            ticket.requiredHeaders.forEach { (k, v) -> builder.header(k, v) }
            if (ticket.requiredHeaders["Content-Type"] == null) {
                builder.header("Content-Type", ticket.contentType.ifBlank { "application/octet-stream" })
            }
            val body = bytes.toRequestBody(mediaType)
            builder.method(method, body)
            signedUploadClient.newCall(builder.build()).execute().use { resp ->
                if (!resp.isSuccessful) error("Signed upload failed (${resp.code})")
            }
            onProgress?.invoke(total, total)
            return@withContext
        }

        val startBuilder = Request.Builder()
            .url(ticket.uploadUrl)
            .header("x-goog-resumable", "start")
        ticket.requiredHeaders.forEach { (k, v) -> startBuilder.header(k, v) }
        if (ticket.requiredHeaders["Content-Type"] == null) {
            startBuilder.header("Content-Type", ticket.contentType.ifBlank { "application/octet-stream" })
        }
        startBuilder.post(ByteArray(0).toRequestBody(mediaType))

        val sessionUrl = signedUploadClient.newCall(startBuilder.build()).execute().use { startResp ->
            if (!startResp.isSuccessful) error("Resumable upload initiation failed (${startResp.code})")
            startResp.header("Location")?.takeIf { it.isNotBlank() } ?: ticket.uploadUrl
        }

        var offset = 0
        while (offset < bytes.size) {
            val end = minOf(offset + RESUMABLE_CHUNK_SIZE, bytes.size)
            val chunkLen = end - offset
            val chunkBody = bytes.toRequestBody(mediaType, offset, chunkLen)
            val chunkReq = Request.Builder()
                .url(sessionUrl)
                .header("Content-Range", "bytes $offset-${end - 1}/${bytes.size}")
                .put(chunkBody)
                .build()
            signedUploadClient.newCall(chunkReq).execute().use { chunkResp ->
                if (!chunkResp.isSuccessful && chunkResp.code != 308) {
                    error("Resumable chunk upload failed (${chunkResp.code})")
                }
            }
            offset = end
            onProgress?.invoke(offset.toLong(), total)
        }
    }

    fun parseJoinRequestConflict(errorJson: String?): JoinRequestConflictPayload? {
        if (errorJson.isNullOrBlank()) return null
        return runCatching {
            val parsed = json.decodeFromString<JoinRequestConflictPayload>(errorJson)
            if (parsed.error == "join_request_pending") parsed else null
        }.getOrNull()
    }

    suspend fun listProjects(): Response<MigrationProjectsResponse> = api.listMigrationProjects()

    suspend fun createProject(
        name: String,
        defaultWarehouseId: String?,
        idempotencyKey: String,
    ): Response<MigrationProject> =
        api.createMigrationProject(
            idempotencyKey = idempotencyKey,
            body = MigrationCreateProjectRequest(
                name = name,
                defaultWarehouseId = defaultWarehouseId?.takeIf { it.isNotBlank() },
            ),
        )

    suspend fun getProjectTree(projectId: String): Response<MigrationProjectTree> =
        api.getMigrationProjectTree(projectId)

    suspend fun createBundle(
        projectId: String,
        files: List<MigrationDeclareFile>,
        idempotencyKey: String,
    ): Response<MigrationCreateBundleResponse> =
        api.createMigrationBundle(projectId, idempotencyKey, MigrationCreateBundleRequest(files))

    suspend fun markBundleUploaded(
        projectId: String,
        bundleId: String,
        artifacts: Map<String, MigrationUploadedArtifactMeta>,
        idempotencyKey: String,
    ): Response<MigrationBundleUploadedResponse> =
        api.markMigrationBundleUploaded(
            projectId = projectId,
            bundleId = bundleId,
            sync = "true",
            idempotencyKey = idempotencyKey,
            body = MigrationBundleUploadedRequest(artifacts),
        )

    suspend fun overrideBundleSort(
        projectId: String,
        bundleId: String,
        request: MigrationBundleSortOverrideRequest,
        idempotencyKey: String,
    ): Response<MigrationBundleSortOverrideResponse> =
        api.overrideMigrationBundleSort(projectId, bundleId, idempotencyKey, request)

    suspend fun getSessionMapping(sessionId: String): Response<MigrationGetSessionMappingResponse> =
        api.getMigrationSessionMapping(sessionId)

    suspend fun getSessionRows(
        sessionId: String,
        limit: Int = 20,
        offset: Int = 0,
        invalidOnly: Boolean = false,
    ): Response<MigrationSessionRowsResponse> =
        api.getMigrationSessionRows(
            sessionId = sessionId,
            limit = limit,
            offset = offset,
            invalidOnly = if (invalidOnly) "1" else null,
        )

    suspend fun confirmSessionMapping(
        sessionId: String,
        request: MigrationConfirmMappingRequest,
        idempotencyKey: String,
    ): Response<MigrationConfirmMappingResponse> =
        api.confirmMigrationSessionMapping(sessionId, idempotencyKey, request)

    suspend fun runDryRun(
        projectId: String,
        sessionId: String?,
        applyMode: String,
        idempotencyKey: String,
    ): Response<MigrationDryRunResponse> =
        if (!sessionId.isNullOrBlank()) {
            api.runMigrationSessionDryRun(sessionId, idempotencyKey, MigrationDryRunRequest(applyMode))
        } else {
            api.runMigrationProjectDryRun(projectId, idempotencyKey, MigrationDryRunRequest(applyMode))
        }

    suspend fun approveSession(
        sessionId: String,
        request: MigrationApproveRequest,
        idempotencyKey: String,
    ): Response<MigrationApproveResponse> =
        api.approveMigrationSession(sessionId, idempotencyKey, request)

    suspend fun applySession(
        sessionId: String,
        applyMode: String,
        chunkSize: Int = 500,
        idempotencyKey: String,
    ): Response<MigrationApplyResult> =
        api.applyMigrationSession(sessionId, idempotencyKey, MigrationApplyRequest(applyMode, chunkSize))

    suspend fun rollbackSession(
        sessionId: String,
        idempotencyKey: String,
    ): Response<MigrationRollbackResult> =
        api.rollbackMigrationSession(sessionId, idempotencyKey)

    suspend fun releaseOpenOrders(
        projectId: String,
        orderIds: List<String>,
        targetStatus: String,
        idempotencyKey: String,
    ): Response<MigrationOpenOrderReleaseResponse> =
        api.releaseMigrationOpenOrders(
            projectId = projectId,
            idempotencyKey = idempotencyKey,
            body = MigrationOpenOrderReleaseRequest(orderIds, targetStatus),
        )

    suspend fun cutoverProject(
        projectId: String,
        idempotencyKey: String,
    ): Response<MigrationCutoverResult> =
        api.cutoverMigrationProject(projectId, idempotencyKey)

    suspend fun lookupINN(inn: String, tenantType: String = "SUPPLIER"): Response<INNLookupResponse> =
        api.lookupINN(INNLookupRequest(inn = inn, tenantType = tenantType))

    suspend fun listJoinRequests(status: String = "PENDING", limit: Int = 100): Response<JoinRequestsResponse> =
        api.listJoinRequests(status = status, limit = limit)

    suspend fun approveJoinRequest(
        requestId: String,
        reason: String,
        idempotencyKey: String,
    ): Response<kotlinx.serialization.json.JsonElement> =
        api.approveJoinRequest(
            requestId = requestId,
            idempotencyKey = idempotencyKey,
            body = if (reason.isBlank()) emptyMap() else mapOf("reason" to reason),
        )

    suspend fun rejectJoinRequest(
        requestId: String,
        reason: String,
        idempotencyKey: String,
    ): Response<kotlinx.serialization.json.JsonElement> =
        api.rejectJoinRequest(
            requestId = requestId,
            idempotencyKey = idempotencyKey,
            body = if (reason.isBlank()) emptyMap() else mapOf("reason" to reason),
        )

    suspend fun listSupplierARInvoices(limit: Int = 100): Response<SupplierARInvoicesResponse> =
        api.listSupplierARInvoices(limit = limit)
}
