package com.pegasusx.driver.data.remote

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import dagger.hilt.android.qualifiers.ApplicationContext
import java.io.ByteArrayOutputStream
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

@Serializable
data class MediaUploadTicket(
    @SerialName("upload_url") val uploadUrl: String,
    @SerialName("public_url") val publicUrl: String? = null,
    @SerialName("image_url") val imageUrl: String? = null,
    @SerialName("content_type") val contentType: String? = null,
) {
    val resolvedPublicUrl: String
        get() = publicUrl?.takeIf { it.isNotBlank() }
            ?: imageUrl?.takeIf { it.isNotBlank() }
            ?: error("upload ticket missing public url")
}

/**
 * Signed GCS PUT for PoD / OS&D evidence photos.
 */
@Singleton
class MediaUploadService @Inject constructor(
    @ApplicationContext private val context: Context,
    private val api: DriverApi,
    private val okHttp: OkHttpClient,
) {
    suspend fun uploadJpegUri(
        uri: Uri,
        purpose: String = "driver_exception",
        orderId: String? = null,
    ): String = withContext(Dispatchers.IO) {
        uploadJpegBytes(compressJpeg(uri), purpose = purpose, orderId = orderId)
    }

    suspend fun uploadJpegBytes(
        bytes: ByteArray,
        purpose: String = "credit_proof",
        orderId: String? = null,
    ): String = withContext(Dispatchers.IO) {
        val ticket = api.getMediaUploadTicket(
            purpose = purpose,
            ext = "jpg",
            orderId = orderId,
        )
        val body = bytes.toRequestBody((ticket.contentType ?: "image/jpeg").toMediaType())
        val request = Request.Builder()
            .url(ticket.uploadUrl)
            .put(body)
            .header("Content-Type", ticket.contentType ?: "image/jpeg")
            .build()
        okHttp.newCall(request).execute().use { resp ->
            if (!resp.isSuccessful) {
                error("gcs_upload_failed:${resp.code}")
            }
        }
        ticket.resolvedPublicUrl
    }

    /** Persist JPEG under app files for offline queue flush. */
    fun savePodJpeg(orderId: String, kind: String, bytes: ByteArray): String {
        val dir = java.io.File(context.filesDir, "pod/$orderId").apply { mkdirs() }
        val file = java.io.File(dir, "$kind.jpg")
        file.writeBytes(bytes)
        return file.absolutePath
    }

    fun readLocalJpeg(path: String): ByteArray = java.io.File(path).readBytes()

    fun compressJpegUri(uri: Uri): ByteArray = compressJpeg(uri)

    private fun compressJpeg(uri: Uri): ByteArray {
        context.contentResolver.openInputStream(uri).use { input ->
            requireNotNull(input) { "cannot_open_image" }
            val bitmap = BitmapFactory.decodeStream(input)
                ?: error("image_decode_failed")
            val stream = ByteArrayOutputStream()
            bitmap.compress(Bitmap.CompressFormat.JPEG, 82, stream)
            if (!bitmap.isRecycled) bitmap.recycle()
            return stream.toByteArray()
        }
    }
}
