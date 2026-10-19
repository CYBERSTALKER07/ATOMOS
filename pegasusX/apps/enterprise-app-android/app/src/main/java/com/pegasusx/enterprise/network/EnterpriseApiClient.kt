package com.pegasusx.enterprise.network

import com.pegasusx.enterprise.model.ClientReplyPayload
import com.pegasusx.enterprise.model.EnterpriseApiResponse
import com.pegasusx.enterprise.model.EnterpriseRequest
import com.pegasusx.enterprise.model.StatusUpdatePayload
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.Date
import java.util.concurrent.TimeUnit

object EnterpriseApiClient {
    private val client = OkHttpClient.Builder()
        .connectTimeout(4, TimeUnit.SECONDS)
        .readTimeout(6, TimeUnit.SECONDS)
        .build()

    private val json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
    }

    // Default to Android Emulator host (10.0.2.2 points to host machine's localhost:3000)
    var baseUrl: String = "http://10.0.2.2:3000"

    private val inMemoryFallback = mutableListOf(
        EnterpriseRequest(
            id = "android-demo-1",
            name = "Rustam Aliev",
            email = "rustam@uzlogistics.com",
            company = "Central Eurasian Freight",
            phone = "+998 90 123 4567",
            role = "Fleet / Dispatch Lead",
            fleet_size = "11–25 sites",
            inquiry_type = "demo",
            subject = "Cross-Border Dispatch Hub",
            message = "Deploying 40 trucks between Tashkent and Samarkand. Need real-time telematics and electronic manifest sync.",
            status = "new",
            priority = "urgent",
            created_at = "2026-10-14T03:00:00Z"
        ),
        EnterpriseRequest(
            id = "android-demo-2",
            name = "Aziza Karimova",
            email = "aziza@agroexpress.uz",
            company = "AgroExpress Perishables",
            phone = "+998 93 987 6543",
            role = "Supplier Operations",
            fleet_size = "4–10 sites",
            inquiry_type = "pilot",
            subject = "Cold Chain Sensor Audit",
            message = "Interested in automated batch traceability and warehouse handover verification.",
            status = "in_review",
            priority = "normal",
            created_at = "2026-10-14T02:15:00Z"
        )
    )

    suspend fun fetchRequests(status: String? = null, inquiryType: String? = null): List<EnterpriseRequest> =
        withContext(Dispatchers.IO) {
            try {
                val urlBuilder = StringBuilder("$baseUrl/api/admin/requests?")
                if (!status.isNullOrBlank() && status != "all") {
                    urlBuilder.append("status=$status&")
                }
                if (!inquiryType.isNullOrBlank() && inquiryType != "all") {
                    urlBuilder.append("inquiry_type=$inquiryType&")
                }

                val request = Request.Builder()
                    .url(urlBuilder.toString().removeSuffix("&").removeSuffix("?"))
                    .get()
                    .build()

                val response = client.newCall(request).execute()
                if (response.isSuccessful) {
                    val body = response.body?.string() ?: return@withContext filterFallback(status, inquiryType)
                    val decoded = json.decodeFromString<EnterpriseApiResponse<List<EnterpriseRequest>>>(body)
                    if (decoded.data != null) {
                        return@withContext decoded.data
                    }
                }
            } catch (e: Exception) {
                // Offline fallback
            }
            filterFallback(status, inquiryType)
        }

    suspend fun sendClientReply(
        requestId: String,
        recipient: String,
        subject: String,
        message: String
    ): Boolean = withContext(Dispatchers.IO) {
        try {
            val mediaType = "application/json; charset=utf-8".toMediaType()
            val payload = ClientReplyPayload(requestId, recipient, subject, message)
            val bodyStr = json.encodeToString(payload)
            val request = Request.Builder()
                .url("$baseUrl/api/admin/reply")
                .post(bodyStr.toRequestBody(mediaType))
                .build()

            val response = client.newCall(request).execute()
            if (response.isSuccessful) {
                val index = inMemoryFallback.indexOfFirst { it.id == requestId }
                if (index != -1) {
                    val item = inMemoryFallback[index]
                    inMemoryFallback[index] = item.copy(
                        status = "contacted",
                        admin_notes = "[Answered at ${Date()}]: $subject\n$message"
                    )
                }
                return@withContext true
            }
        } catch (e: Exception) {
            // Offline fallback
        }

        val index = inMemoryFallback.indexOfFirst { it.id == requestId }
        if (index != -1) {
            val item = inMemoryFallback[index]
            inMemoryFallback[index] = item.copy(
                status = "contacted",
                admin_notes = "[Drafted Offline]: $subject\n$message"
            )
        }
        true
    }

    suspend fun updateStatus(id: String, status: String, notes: String? = null): Boolean =
        withContext(Dispatchers.IO) {
            try {
                val mediaType = "application/json; charset=utf-8".toMediaType()
                val bodyStr = json.encodeToString(StatusUpdatePayload(id, status, notes))
                val request = Request.Builder()
                    .url("$baseUrl/api/admin/requests")
                    .patch(bodyStr.toRequestBody(mediaType))
                    .build()

                val response = client.newCall(request).execute()
                if (response.isSuccessful) {
                    // Succeeded
                }
            } catch (e: Exception) {
                // Offline
            }

            val index = inMemoryFallback.indexOfFirst { it.id == id }
            if (index != -1) {
                val item = inMemoryFallback[index]
                inMemoryFallback[index] = item.copy(
                    status = status,
                    admin_notes = notes ?: item.admin_notes
                )
            }
            true
        }

    suspend fun deleteRequest(id: String): Boolean =
        withContext(Dispatchers.IO) {
            try {
                val request = Request.Builder()
                    .url("$baseUrl/api/admin/requests?id=$id")
                    .delete()
                    .build()
                client.newCall(request).execute()
            } catch (e: Exception) {
                // Offline
            }
            inMemoryFallback.removeAll { it.id == id }
            true
        }

    private fun filterFallback(status: String?, inquiryType: String?): List<EnterpriseRequest> {
        var list = inMemoryFallback.toList()
        if (!status.isNullOrBlank() && status != "all") {
            list = list.filter { it.status == status }
        }
        if (!inquiryType.isNullOrBlank() && inquiryType != "all") {
            list = list.filter { it.inquiry_type == inquiryType }
        }
        return list
    }
}
