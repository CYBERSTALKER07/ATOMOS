package com.pegasusx.retailer.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class RetailerClaimLine(
    val sku: String,
    val quantity: Long,
    val reason: String? = null,
    @SerialName("unit_price_minor") val unitPriceMinor: Long? = null,
    @SerialName("amount_minor") val amountMinor: Long? = null,
)

@Serializable
data class RetailerClaim(
    @SerialName("claim_id") val claimId: String,
    @SerialName("order_id") val orderId: String = "",
    @SerialName("claim_type") val claimType: String = "",
    val status: String = "",
    val description: String? = null,
    @SerialName("amount_minor") val amountMinor: Long? = null,
    val currency: String? = null,
    @SerialName("line_items") val lineItems: List<RetailerClaimLine>? = null,
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class RetailerClaimsListResponse(
    val claims: List<RetailerClaim> = emptyList(),
)

@Serializable
data class ClaimEligibility(
    val eligible: Boolean = false,
    @SerialName("ends_at") val endsAt: String? = null,
    @SerialName("window_hours") val windowHours: Int = 0,
    @SerialName("hours_remaining") val hoursRemaining: Double = 0.0,
    @SerialName("policy_source") val policySource: String = "",
    @SerialName("photo_required_types") val photoRequiredTypes: List<String> = emptyList(),
    @SerialName("order_status") val orderStatus: String = "",
    val reason: String = "",
)

@Serializable
data class FileClaimEvidenceBody(
    @SerialName("evidence_type") val evidenceType: String,
    val uri: String,
    @SerialName("mime_type") val mimeType: String = "image/jpeg",
)

@Serializable
data class FileClaimLineBody(
    val sku: String,
    val quantity: Long,
    val reason: String,
)

@Serializable
data class FileClaimRequestBody(
    @SerialName("claim_type") val claimType: String,
    val description: String = "",
    @SerialName("line_items") val lineItems: List<FileClaimLineBody> = emptyList(),
    val evidences: List<FileClaimEvidenceBody> = emptyList(),
)

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
