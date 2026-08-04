package com.pegasusx.retailer.ui.components

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CameraAlt
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.pegasusx.retailer.data.api.MediaUploadService
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.ClaimEligibility
import com.pegasusx.retailer.data.model.FileClaimEvidenceBody
import com.pegasusx.retailer.data.model.FileClaimLineBody
import com.pegasusx.retailer.data.model.FileClaimRequestBody
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.RetailerClaim
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import kotlinx.coroutines.launch

private val claimTypes = listOf(
    "CONCEALED_DAMAGE", "DAMAGED", "MISSING", "TAMPER", "TEMPERATURE", "OTHER",
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FileClaimSheet(
    order: Order,
    api: PegasusApi,
    mediaUpload: MediaUploadService,
    onDismiss: () -> Unit,
    onFiled: (RetailerClaim) -> Unit = {},
    preferredSku: String? = null,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    var claimType by remember { mutableStateOf("CONCEALED_DAMAGE") }
    var description by remember { mutableStateOf("") }
    val selectedQty = remember { mutableStateMapOf<String, Int>() }
    var photoUrl by remember { mutableStateOf("") }
    var previewUri by remember { mutableStateOf<String?>(null) }
    var isUploading by remember { mutableStateOf(false) }
    var isSubmitting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var skuWarning by remember { mutableStateOf<String?>(null) }
    var existing by remember { mutableStateOf<List<RetailerClaim>>(emptyList()) }
    var successId by remember { mutableStateOf<String?>(null) }
    var eligibility by remember { mutableStateOf<ClaimEligibility?>(null) }
    var eligLoading by remember { mutableStateOf(true) }

    val photoPicker = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.GetContent(),
    ) { uri: Uri? ->
        if (uri == null) return@rememberLauncherForActivityResult
        previewUri = uri.toString()
        scope.launch {
            isUploading = true
            error = null
            try {
                photoUrl = mediaUpload.uploadJpegUri(
                    uri = uri,
                    purpose = "claim_evidence",
                    orderId = order.id,
                )
            } catch (e: Exception) {
                photoUrl = ""
                error = e.message ?: "Photo upload failed"
            } finally {
                isUploading = false
            }
        }
    }

    LaunchedEffect(order.id) {
        existing = runCatching { api.listOrderClaims(order.id).claims }.getOrDefault(emptyList())
        eligLoading = true
        eligibility = runCatching { api.getClaimEligibility(order.id) }.getOrNull()
        eligLoading = false
    }

    LaunchedEffect(order.id, preferredSku) {
        val sku = preferredSku?.trim().orEmpty()
        if (sku.isEmpty()) {
            skuWarning = null
            return@LaunchedEffect
        }
        val match = order.items.firstOrNull {
            it.productId == sku || it.id == sku
        }
        if (match == null) {
            skuWarning = "SKU $sku is not on this order — pick another line."
            return@LaunchedEffect
        }
        val key = match.productId.ifBlank { match.id }
        selectedQty[key] = minOf(1, match.quantity)
        skuWarning = null
    }

    val needsPhoto = claimType in setOf("DAMAGED", "CONCEALED_DAMAGE", "TAMPER", "TEMPERATURE")
    val totalSelected = selectedQty.values.sum()

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = MaterialTheme.colorScheme.surface,
        shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
    ) {
        LazyColumn(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                Text(
                    "File claim · #${order.id.takeLast(6)}",
                    style = MaterialTheme.typography.titleLarge,
                    fontWeight = FontWeight.Bold,
                )
                when {
                    eligLoading -> Text(
                        "Checking claim window…",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.55f),
                    )
                    eligibility?.eligible == true -> Text(
                        "Eligible until ${formatClaimEndsAt(eligibility?.endsAt)} (${eligibility?.hoursRemaining}h left). Amounts use order prices.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.55f),
                    )
                    eligibility != null -> Text(
                        "Window closed" + when (eligibility?.reason) {
                            "claim_window_expired" -> " — filing deadline passed."
                            "order_not_completed" -> " — order not COMPLETED yet."
                            else -> "."
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                    else -> Text(
                        "Within 48 hours of delivery (server enforces). Amounts use order prices.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.55f),
                    )
                }
                skuWarning?.let {
                    Spacer(Modifier.height(6.dp))
                    Text(
                        it,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.tertiary,
                    )
                }
            }

            item {
                Text("Claim type", fontWeight = FontWeight.SemiBold)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    claimTypes.take(3).forEach { t ->
                        FilterChip(
                            selected = claimType == t,
                            onClick = { claimType = t },
                            label = { Text(t.replace('_', ' ').lowercase().replaceFirstChar { it.uppercase() }, maxLines = 1) },
                        )
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    claimTypes.drop(3).forEach { t ->
                        FilterChip(
                            selected = claimType == t,
                            onClick = { claimType = t },
                            label = { Text(t.replace('_', ' ').lowercase().replaceFirstChar { it.uppercase() }, maxLines = 1) },
                        )
                    }
                }
            }

            item {
                Text("Items", fontWeight = FontWeight.SemiBold)
            }
            items(order.items, key = { it.id }) { item ->
                val sku = item.productId.ifBlank { item.id }
                val qty = selectedQty[sku] ?: 0
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(item.productName, style = MaterialTheme.typography.bodyMedium)
                        Text("SKU $sku · ordered ${item.quantity}", style = MaterialTheme.typography.labelSmall)
                    }
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        TextButton(
                            onClick = { if (qty > 0) selectedQty[sku] = qty - 1 },
                            enabled = qty > 0 && !isSubmitting,
                        ) { Text("−") }
                        Text("$qty", fontWeight = FontWeight.Bold, modifier = Modifier.width(28.dp))
                        TextButton(
                            onClick = {
                                if (qty < item.quantity) selectedQty[sku] = qty + 1
                            },
                            enabled = qty < item.quantity && !isSubmitting,
                        ) { Text("+") }
                    }
                }
            }

            item {
                Text("Photo proof", fontWeight = FontWeight.SemiBold)
                OutlinedButton(
                    onClick = { photoPicker.launch("image/*") },
                    enabled = !isSubmitting && !isUploading,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (isUploading) {
                        CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                        Spacer(Modifier.width(8.dp))
                        Text("Uploading…")
                    } else {
                        Icon(Icons.Filled.CameraAlt, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(8.dp))
                        Text(if (photoUrl.isBlank()) "Take or choose photo" else "Photo ready — change")
                    }
                }
                previewUri?.let { uri ->
                    AsyncImage(
                        model = uri,
                        contentDescription = "Claim proof",
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(140.dp)
                            .clip(RoundedCornerShape(12.dp)),
                        contentScale = ContentScale.Crop,
                    )
                }
                if (needsPhoto) {
                    Text(
                        "Required for this claim type.",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
            }

            item {
                OutlinedTextField(
                    value = description,
                    onValueChange = { description = it },
                    label = { Text("What happened?") },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                    enabled = !isSubmitting,
                )
            }

            error?.let { msg ->
                item {
                    Text(msg, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
            }
            successId?.let { id ->
                item {
                    Text("Claim filed: $id", color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
                }
            }

            if (existing.isNotEmpty()) {
                item {
                    Text("Previous claims", fontWeight = FontWeight.SemiBold)
                }
                items(existing, key = { it.claimId }) { c ->
                    Text(
                        "${c.claimType} · ${c.status} · ${c.amountMinor ?: 0} ${c.currency ?: "UZS"}",
                        style = MaterialTheme.typography.bodySmall,
                    )
                }
            }

            item {
                Button(
                    onClick = {
                        scope.launch {
                            error = null
                            if (eligibility?.eligible == false) {
                                error = "Window closed — claim window has expired."
                                return@launch
                            }
                            if (totalSelected <= 0) {
                                error = "Select at least one item quantity."
                                return@launch
                            }
                            if (needsPhoto && photoUrl.isBlank()) {
                                error = "Photo required for this claim type."
                                return@launch
                            }
                            isSubmitting = true
                            try {
                                val lines = selectedQty.mapNotNull { (sku, qty) ->
                                    if (qty <= 0) null
                                    else FileClaimLineBody(
                                        sku = sku,
                                        quantity = qty.toLong(),
                                        reason = if (claimType == "MISSING") "MISSING" else "DAMAGED",
                                    )
                                }
                                val evidences = if (photoUrl.isNotBlank()) {
                                    listOf(
                                        FileClaimEvidenceBody(
                                            evidenceType = "PHOTO",
                                            uri = photoUrl,
                                            mimeType = "image/jpeg",
                                        ),
                                    )
                                } else {
                                    emptyList()
                                }
                                val body = FileClaimRequestBody(
                                    claimType = claimType,
                                    description = description,
                                    lineItems = lines,
                                    evidences = evidences,
                                )
                                val fingerprint = buildString {
                                    append(claimType).append('|').append(description).append('|')
                                    lines.sortedBy { it.sku }.forEach {
                                        append(it.sku).append(':').append(it.quantity).append(';')
                                    }
                                    evidences.forEach { append(it.uri).append(';') }
                                }
                                val claim = api.fileOrderClaim(
                                    orderId = order.id,
                                    body = body,
                                    idempotencyKey = com.pegasusx.retailer.util.RetailerIdempotencyKeys.claimFile(
                                        order.id,
                                        fingerprint,
                                    ),
                                )
                                successId = claim.claimId
                                existing = runCatching { api.listOrderClaims(order.id).claims }
                                    .getOrDefault(existing)
                                onFiled(claim)
                            } catch (e: Exception) {
                                error = e.message ?: "Claim failed"
                            } finally {
                                isSubmitting = false
                            }
                        }
                    },
                    enabled = !isSubmitting && !isUploading && totalSelected > 0 &&
                        !eligLoading && (eligibility == null || eligibility?.eligible == true),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(bottom = 28.dp),
                ) {
                    if (isSubmitting) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(18.dp),
                            strokeWidth = 2.dp,
                            color = MaterialTheme.colorScheme.onPrimary,
                        )
                        Spacer(Modifier.width(8.dp))
                    }
                    Text("Submit claim", fontWeight = FontWeight.Bold)
                }
            }
        }
    }
}

internal fun formatClaimEndsAt(endsAt: String?): String {
    if (endsAt.isNullOrBlank()) return "window end"
    return runCatching {
        val instant = Instant.parse(endsAt)
        DateTimeFormatter.ofLocalizedDateTime(FormatStyle.SHORT)
            .withZone(ZoneId.systemDefault())
            .format(instant)
    }.getOrDefault(endsAt)
}
