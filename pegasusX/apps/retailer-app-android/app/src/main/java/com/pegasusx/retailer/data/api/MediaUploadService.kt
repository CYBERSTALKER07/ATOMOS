package com.pegasusx.retailer.data.api

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.net.Uri
import com.pegasusx.retailer.data.model.MediaUploadTicket
import dagger.hilt.android.qualifiers.ApplicationContext
import java.io.ByteArrayOutputStream
import javax.inject.Inject
import javax.inject.Singleton
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

/** Signed GCS PUT for claim evidence photos. */
@Singleton
class MediaUploadService @Inject constructor(
    @ApplicationContext private val context: Context,
    private val api: PegasusApi,
    private val okHttp: OkHttpClient,
) {
    suspend fun uploadJpegUri(
        uri: Uri,
        purpose: String = "claim_evidence",
        orderId: String? = null,
    ): String = withContext(Dispatchers.IO) {
        val bytes = compressJpeg(uri)
        val ticket: MediaUploadTicket = api.getMediaUploadTicket(
            purpose = purpose,
            ext = "jpg",
            orderId = orderId,
        )
        val contentType = ticket.contentType ?: "image/jpeg"
        val body = bytes.toRequestBody(contentType.toMediaType())
        val request = Request.Builder()
            .url(ticket.uploadUrl)
            .put(body)
            .header("Content-Type", contentType)
            .build()
        okHttp.newCall(request).execute().use { resp ->
            if (!resp.isSuccessful) error("gcs_upload_failed:${resp.code}")
        }
        ticket.resolvedPublicUrl
    }

    private fun compressJpeg(uri: Uri): ByteArray {
        context.contentResolver.openInputStream(uri).use { input ->
            requireNotNull(input) { "cannot_open_image" }
            val bitmap = BitmapFactory.decodeStream(input) ?: error("image_decode_failed")
            val stream = ByteArrayOutputStream()
            bitmap.compress(Bitmap.CompressFormat.JPEG, 82, stream)
            if (!bitmap.isRecycled) bitmap.recycle()
            return stream.toByteArray()
        }
    }
}
