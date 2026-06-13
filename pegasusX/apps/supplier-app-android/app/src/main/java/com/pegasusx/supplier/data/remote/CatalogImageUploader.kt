package com.pegasusx.supplier.data.remote

import android.content.Context
import android.net.Uri
import com.pegasusx.supplier.data.model.CatalogUploadTicket
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

object CatalogImageUploader {
    private val client = OkHttpClient()

    suspend fun uploadTicketImage(
        api: SupplierApi,
        context: Context,
        uri: Uri,
        ext: String,
    ): Result<String> = withContext(Dispatchers.IO) {
        runCatching {
            val ticketResp = api.getCatalogUploadTicket(ext)
            if (!ticketResp.isSuccessful) {
                error("upload ticket ${ticketResp.code()}")
            }
            val ticket = ticketResp.body() ?: error("empty upload ticket")
            if (!ticket.uploadUrl.contains("placehold.co")) {
                val bytes = context.contentResolver.openInputStream(uri)?.use { it.readBytes() }
                    ?: error("cannot read image")
                val contentType = when (ext.lowercase()) {
                    "png" -> "image/png"
                    "webp" -> "image/webp"
                    else -> "image/jpeg"
                }
                val request = Request.Builder()
                    .url(ticket.uploadUrl)
                    .put(bytes.toRequestBody(contentType.toMediaType()))
                    .build()
                client.newCall(request).execute().use { response ->
                    if (!response.isSuccessful) error("storage upload ${response.code}")
                }
            }
            ticket.imageUrl
        }
    }

    fun fileExtension(fileName: String?): String {
        val ext = fileName?.substringAfterLast('.', "")?.lowercase().orEmpty()
        return ext.ifBlank { "jpg" }
    }
}
