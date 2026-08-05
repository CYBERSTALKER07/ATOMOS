package com.pegasusx.driver.ui.screens.offload

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.AddCircle
import androidx.compose.material.icons.filled.CameraAlt
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.RemoveCircle
import androidx.compose.material.icons.filled.RemoveCircleOutline
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.pegasusx.driver.data.model.ConfirmOffloadResponse
import com.pegasusx.driver.data.model.RejectionReason
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusOrange
import com.pegasusx.driver.ui.theme.StatusRed
import com.pegasusx.driver.ui.theme.StatusBlue
import com.pegasusx.driver.ui.theme.formattedAmount

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun OffloadReviewScreen(
    onClose: () -> Unit,
    onOffloadConfirmed: (ConfirmOffloadResponse) -> Unit,
    onShopClosed: (String) -> Unit = {},
    onCreditDelivery: (String) -> Unit = {},
    onReportMissing: (String) -> Unit = {},
    viewModel: OffloadReviewViewModel = hiltViewModel()
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current
    val photoPicker = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.GetContent(),
    ) { uri ->
        if (uri != null) viewModel.uploadEvidencePhoto(uri)
    }

    // If offload confirmed, route to next screen
    state.offloadResult?.let { result ->
        onOffloadConfirmed(result)
        return
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
    ) {
        // Header
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 56.dp, start = 8.dp, end = 16.dp, bottom = 8.dp)
        ) {
            IconButton(onClick = onClose) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = lab.fg)
            }
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "OFFLOAD REVIEW",
                    fontSize = 10.sp,
                    fontWeight = FontWeight.Black,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgTertiary,
                    letterSpacing = 1.2.sp
                )
                Text(
                    text = state.retailerName.ifBlank { "Loading..." },
                    fontSize = 20.sp,
                    fontWeight = FontWeight.Bold,
                    color = lab.fg
                )
            }
        }

        // Totals bar
        com.pegasusx.driver.ui.screens.offload.components.OffloadSummaryCard(
            originalTotal = state.originalTotal,
            adjustedTotal = state.adjustedTotal,
            hasRejections = state.hasRejections
        )

        // Line items
        LazyColumn(
            modifier = Modifier
                .weight(1f)
                .padding(horizontal = 16.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            itemsIndexed(state.audits) { index, audit ->
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 8.dp)
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Icon(
                            imageVector = if (audit.isFullyRejected) Icons.Default.RemoveCircleOutline else Icons.Default.CheckCircle,
                            contentDescription = null,
                            tint = when {
                                audit.isFullyRejected -> StatusRed
                                audit.isPartiallyRejected -> StatusOrange
                                else -> StatusGreen
                            },
                            modifier = Modifier.size(20.dp)
                        )
                        Spacer(modifier = Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = audit.item.productName,
                                fontSize = 14.sp,
                                fontWeight = FontWeight.Medium,
                                color = lab.fg,
                                textDecoration = if (audit.isFullyRejected) TextDecoration.LineThrough else null
                            )
                            Text(
                                text = "${audit.item.quantity}× · ${audit.item.unitPrice.formattedAmount()}/ea",
                                fontSize = 11.sp,
                                color = lab.fgTertiary
                            )
                        }
                        Text(
                            text = audit.acceptedTotal.formattedAmount(),
                            fontSize = 13.sp,
                            fontWeight = FontWeight.Bold,
                            color = when {
                                audit.isFullyRejected -> lab.fgTertiary
                                audit.isPartiallyRejected -> StatusOrange
                                else -> lab.fg
                            }
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        // +/− stepper for rejected quantity
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(2.dp)
                        ) {
                            IconButton(
                                onClick = { viewModel.updateRejectedQty(index, -1) },
                                modifier = Modifier.size(32.dp),
                                enabled = audit.rejected > 0
                            ) {
                                Icon(
                                    imageVector = Icons.Default.RemoveCircle,
                                    contentDescription = "Reduce rejected",
                                    tint = if (audit.rejected > 0) StatusRed else lab.fgTertiary,
                                    modifier = Modifier.size(22.dp)
                                )
                            }
                            Text(
                                text = audit.rejected.toString(),
                                fontSize = 14.sp,
                                fontWeight = FontWeight.Bold,
                                fontFamily = FontFamily.Monospace,
                                color = when {
                                    audit.isFullyRejected -> StatusRed
                                    audit.isPartiallyRejected -> StatusOrange
                                    else -> StatusGreen
                                },
                                modifier = Modifier.width(22.dp),
                                textAlign = androidx.compose.ui.text.style.TextAlign.Center
                            )
                            IconButton(
                                onClick = { viewModel.updateRejectedQty(index, 1) },
                                modifier = Modifier.size(32.dp),
                                enabled = audit.rejected < audit.item.quantity
                            ) {
                                Icon(
                                    imageVector = Icons.Default.AddCircle,
                                    contentDescription = "Increase rejected",
                                    tint = if (audit.rejected < audit.item.quantity) StatusRed else lab.fgTertiary,
                                    modifier = Modifier.size(22.dp)
                                )
                            }
                        }
                    }

                    if (audit.rejected > 0) {
                        Spacer(modifier = Modifier.height(8.dp))
                        FlowRow(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(start = 32.dp),
                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                            verticalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            RejectionReason.values().forEach { reason ->
                                FilterChip(
                                    selected = audit.reason == reason,
                                    onClick = { viewModel.updateReason(index, reason) },
                                    label = {
                                        Text(
                                            text = reason.label,
                                            fontSize = 12.sp,
                                            fontWeight = FontWeight.Medium
                                        )
                                    }
                                )
                            }
                        }
                        if (audit.reason == RejectionReason.OTHER) {
                            OutlinedTextField(
                                value = audit.customReason,
                                onValueChange = { viewModel.updateCustomReason(index, it) },
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(start = 32.dp, top = 8.dp),
                                placeholder = { Text("Describe the issue") },
                                singleLine = false,
                                minLines = 2,
                                colors = TextFieldDefaults.colors(
                                    focusedContainerColor = lab.card,
                                    unfocusedContainerColor = lab.card,
                                ),
                            )
                        }
                    }
                }
                if (index < state.audits.lastIndex) {
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant, thickness = 1.dp)
                }
            }
        }

        // PoD / damage photo proof (required for credit leave; also for DAMAGED / WRONG_ITEM)
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
                Text(
                    text = "PROOF OF DELIVERY PHOTO",
                    fontSize = 10.sp,
                    fontWeight = FontWeight.Black,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fgTertiary,
                    letterSpacing = 1.2.sp,
                )
                androidx.compose.material3.OutlinedButton(
                    onClick = { photoPicker.launch("image/*") },
                    enabled = !state.isSubmitting && !state.isUploadingPhoto,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(44.dp),
                    shape = MaterialTheme.shapes.medium,
                    colors = ButtonDefaults.outlinedButtonColors(contentColor = lab.fg),
                ) {
                    if (state.isUploadingPhoto) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(18.dp),
                            strokeWidth = 2.dp,
                            color = lab.fg,
                        )
                        Spacer(Modifier.width(8.dp))
                        Text("Uploading…", fontSize = 14.sp)
                    } else {
                        Icon(Icons.Filled.CameraAlt, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(8.dp))
                        Text(
                            text = if (state.evidencePhotoUrl.isBlank()) {
                                "Take or choose photo"
                            } else {
                                "Photo ready — change"
                            },
                            fontWeight = FontWeight.Medium,
                            fontSize = 14.sp,
                        )
                    }
                }
                state.photoPreviewUri?.let { uri ->
                    AsyncImage(
                        model = uri,
                        contentDescription = "Damage proof",
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(140.dp)
                            .clip(RoundedCornerShape(12.dp)),
                        contentScale = ContentScale.Crop,
                    )
                }
            Text(
                text = "Required for credit leave and for damaged or wrong-item rejections.",
                fontSize = 11.sp,
                color = lab.fgTertiary,
            )
        }

        // Error
        state.error?.let { error ->
            Text(
                text = error,
                color = StatusRed,
                fontSize = 12.sp,
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)
            )
        }

        com.pegasusx.driver.ui.screens.offload.components.OffloadActionFooter(
            isSubmitting = state.isSubmitting,
            isUploadingPhoto = state.isUploadingPhoto,
            hasRejections = state.hasRejections,
            orderId = state.orderId,
            onShopClosed = onShopClosed,
            onCreditDelivery = { viewModel.markCreditDelivery() },
            onReportMissing = onReportMissing,
            onConfirm = { viewModel.confirmOffload() }
        )
    }
}
