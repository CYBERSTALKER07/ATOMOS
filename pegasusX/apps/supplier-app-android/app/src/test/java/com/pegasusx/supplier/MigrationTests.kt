package com.pegasusx.supplier

import com.pegasusx.supplier.data.model.INNLookupResponse
import com.pegasusx.supplier.data.model.MigrationStagedRow
import com.pegasusx.supplier.data.model.MigrationUploadTicket
import com.pegasusx.supplier.data.remote.MigrationRepository
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierWSMessage
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Protocol
import okhttp3.Response
import okhttp3.ResponseBody.Companion.toResponseBody
import okio.Buffer
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.lang.reflect.Proxy

class MigrationTests {
    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    private fun unusedApi(): SupplierApi = Proxy.newProxyInstance(
        SupplierApi::class.java.classLoader,
        arrayOf(SupplierApi::class.java),
    ) { _, method, _ ->
        error("Unexpected SupplierApi call in unit test: ${method.name}")
    } as SupplierApi

    @Test
    fun sha256HexMatchesKnownVector() {
        val digest = MigrationRepository.sha256Hex("abc".toByteArray(Charsets.UTF_8))
        assertEquals(
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
            digest,
        )
    }

    @Test
    fun directSignedPutUploadSendsRequiredHeadersWithoutAuth() = runBlocking {
        val repo = MigrationRepository(unusedApi(), json)
        var recordedMethod: String? = null
        var recordedAuth: String? = "present"
        var recordedContentType: String? = null
        var recordedRangeHeader: String? = null
        var recordedBody: ByteArray = ByteArray(0)

        val client = OkHttpClient.Builder()
            .addInterceptor(
                Interceptor { chain ->
                    val req = chain.request()
                    recordedMethod = req.method
                    recordedAuth = req.header("Authorization")
                    recordedContentType = req.header("Content-Type")
                    recordedRangeHeader = req.header("x-goog-content-length-range")
                    val buf = Buffer()
                    req.body?.writeTo(buf)
                    recordedBody = buf.readByteArray()
                    Response.Builder()
                        .request(req)
                        .protocol(Protocol.HTTP_1_1)
                        .code(200)
                        .message("OK")
                        .body("".toResponseBody("text/plain".toMediaType()))
                        .build()
                },
            )
            .build()
        repo.setSignedUploadClientForTest(client)

        val payload = "sku,name,price_minor\nSKU-1,Tea,12000\n".toByteArray()
        val ticket = MigrationUploadTicket(
            artifactId = "art-1",
            fileName = "catalog.csv",
            uploadUrl = "https://storage.googleapis.com/pegasus-bucket/art-1?sig=1",
            method = "PUT",
            contentType = "text/csv",
            resumable = false,
            requiredHeaders = mapOf("x-goog-content-length-range" to "0,1048576"),
        )

        var lastProgress = 0L to 0L
        repo.uploadArtifactToSignedTicket(ticket, payload) { sent, total ->
            lastProgress = sent to total
        }

        assertEquals("PUT", recordedMethod)
        assertNull(recordedAuth)
        assertEquals("text/csv", recordedContentType)
        assertEquals("0,1048576", recordedRangeHeader)
        assertArrayEquals(payload, recordedBody)
        assertEquals(payload.size.toLong() to payload.size.toLong(), lastProgress)
    }

    @Test
    fun resumableUploadOver8MiBSplitsChunksAndReassemblesExactBytes() = runBlocking {
        val repo = MigrationRepository(unusedApi(), json)
        val totalSize = MigrationRepository.RESUMABLE_CHUNK_SIZE + 123_456
        val payload = ByteArray(totalSize) { idx -> ((idx * 31) and 0xFF).toByte() }
        val reassembled = Buffer()
        val seenRanges = mutableListOf<String>()
        var startSeen = false
        var callIndex = 0

        val client = OkHttpClient.Builder()
            .addInterceptor(
                Interceptor { chain ->
                    val req = chain.request()
                    callIndex++
                    when (callIndex) {
                        1 -> {
                            assertEquals("POST", req.method)
                            assertEquals("start", req.header("x-goog-resumable"))
                            startSeen = true
                            Response.Builder()
                                .request(req)
                                .protocol(Protocol.HTTP_1_1)
                                .code(200)
                                .message("OK")
                                .header("Location", "https://storage.googleapis.com/upload/session-abc")
                                .body("".toResponseBody("text/plain".toMediaType()))
                                .build()
                        }
                        2 -> {
                            assertEquals("PUT", req.method)
                            assertEquals("https://storage.googleapis.com/upload/session-abc", req.url.toString())
                            seenRanges.add(req.header("Content-Range").orEmpty())
                            req.body?.writeTo(reassembled)
                            Response.Builder()
                                .request(req)
                                .protocol(Protocol.HTTP_1_1)
                                .code(308)
                                .message("Resume Incomplete")
                                .body("".toResponseBody("text/plain".toMediaType()))
                                .build()
                        }
                        else -> {
                            assertEquals("PUT", req.method)
                            seenRanges.add(req.header("Content-Range").orEmpty())
                            req.body?.writeTo(reassembled)
                            Response.Builder()
                                .request(req)
                                .protocol(Protocol.HTTP_1_1)
                                .code(200)
                                .message("OK")
                                .body("".toResponseBody("text/plain".toMediaType()))
                                .build()
                        }
                    }
                },
            )
            .build()
        repo.setSignedUploadClientForTest(client)

        val ticket = MigrationUploadTicket(
            artifactId = "art-big",
            fileName = "big_ledger.xlsx",
            uploadUrl = "https://storage.googleapis.com/pegasus-bucket/art-big",
            method = "PUT",
            contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            resumable = true,
        )

        val progressMarks = mutableListOf<Long>()
        repo.uploadArtifactToSignedTicket(ticket, payload) { sent, _ ->
            progressMarks.add(sent)
        }

        assertTrue(startSeen)
        assertEquals(2, seenRanges.size)
        assertEquals("bytes 0-${MigrationRepository.RESUMABLE_CHUNK_SIZE - 1}/$totalSize", seenRanges[0])
        assertEquals("bytes ${MigrationRepository.RESUMABLE_CHUNK_SIZE}-${totalSize - 1}/$totalSize", seenRanges[1])
        assertArrayEquals(payload, reassembled.readByteArray())
        assertEquals(listOf(MigrationRepository.RESUMABLE_CHUNK_SIZE.toLong(), totalSize.toLong()), progressMarks)
    }

    @Test
    fun wsMessageParsesFlatAndNestedMigrationApplyProgress() {
        val flatRaw = """
            {"type":"MIGRATION_APPLY_PROGRESS","session_id":"sess-1","processed_rows":500,"total_rows":1200,"chunk_id":"chk-1"}
        """.trimIndent()
        val flatEvt = json.decodeFromString<SupplierWSMessage>(flatRaw).migrationApplyProgress()
        assertNotNull(flatEvt)
        assertEquals("sess-1", flatEvt!!.sessionId)
        assertEquals(500L, flatEvt.processedRows)
        assertEquals(1200L, flatEvt.totalRows)
        assertEquals("chk-1", flatEvt.chunkId)

        val nestedRaw = """
            {"type":"MIGRATION_APPLY_PROGRESS","payload":{"session_id":"sess-2","processed_rows":900,"total_rows":900,"chunk_id":"chk-last"}}
        """.trimIndent()
        val nestedEvt = json.decodeFromString<SupplierWSMessage>(nestedRaw).migrationApplyProgress()
        assertNotNull(nestedEvt)
        assertEquals("sess-2", nestedEvt!!.sessionId)
        assertEquals(900L, nestedEvt.processedRows)
        assertEquals(900L, nestedEvt.totalRows)
        assertEquals("chk-last", nestedEvt.chunkId)

        val otherRaw = """{"type":"ORDER_CREATED"}"""
        assertNull(json.decodeFromString<SupplierWSMessage>(otherRaw).migrationApplyProgress())
    }

    @Test
    fun migrationIdempotencyKeysAreDeterministic() {
        val k1 = SupplierIdempotencyKeys.migrationOpenOrdersRelease("proj-1", listOf("o-2", "o-1"), "PENDING")
        val k2 = SupplierIdempotencyKeys.migrationOpenOrdersRelease("proj-1", listOf("o-1", "o-2"), "PENDING")
        assertEquals(k1, k2)
        assertTrue(k1.startsWith("migration-open-orders-release:proj-1:PENDING:"))

        assertEquals(
            "migration-session-rollback:sess-9",
            SupplierIdempotencyKeys.migrationSessionRollback("sess-9"),
        )
        assertEquals(
            "migration-cutover:proj-9",
            SupplierIdempotencyKeys.migrationCutover("proj-9"),
        )
        assertEquals(
            "join-request-decide:req-1:approve",
            SupplierIdempotencyKeys.joinRequestDecide("req-1", "APPROVE"),
        )
    }

    @Test
    fun joinRequestConflictAndInnLookupAndStagedRowDecodeCleanly() {
        val repo = MigrationRepository(unusedApi(), json)
        val conflict = repo.parseJoinRequestConflict(
            """{"error":"join_request_pending","request_id":"req-77","target_tenant_id":"sup-1","legal_name":"OOO Samarkand","normalized_inn":"301234567"}""",
        )
        assertNotNull(conflict)
        assertEquals("req-77", conflict!!.requestId)
        assertEquals("301234567", conflict.normalizedInn)

        val unavailable = json.decodeFromString<INNLookupResponse>(
            """{"status":"UNAVAILABLE","normalized_inn":"301234567"}""",
        )
        assertEquals("UNAVAILABLE", unavailable.status)
        assertNull(unavailable.legalName)

        val row = MigrationStagedRow(
            rowIndex = 12L,
            rawData = mapOf(
                "sku" to JsonPrimitive("SKU-42"),
                "qty" to JsonPrimitive(150),
                "cold" to JsonPrimitive(true),
                "note" to JsonNull,
            ),
        )
        assertEquals("SKU-42", row.textValue("sku"))
        assertEquals("150", row.textValue("qty"))
        assertEquals("true", row.textValue("cold"))
        assertEquals("", row.textValue("note"))
        assertEquals("", row.textValue("missing"))
    }
}
