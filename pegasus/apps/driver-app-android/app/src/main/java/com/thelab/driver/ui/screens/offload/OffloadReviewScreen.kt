package com.pegasus.driver.ui.screens.offload

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
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
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.RemoveCircle
import androidx.compose.material.icons.filled.RemoveCircleOutline
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.driver.data.model.ConfirmOffloadResponse
import com.pegasus.driver.ui.theme.LocalLabColors
import com.pegasus.driver.ui.theme.StatusGreen
import com.pegasus.driver.ui.theme.StatusOrange
import com.pegasus.driver.ui.theme.StatusRed
import com.pegasus.driver.ui.theme.StatusBlue
import com.pegasus.driver.ui.theme.formattedAmount

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
    val lab = LocalLabColors.current

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
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(lab.card)
                .padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceBetween
        ) {
            Column {
                Text("Original", fontSize = 10.sp, color = lab.fgTertiary, fontFamily = FontFamily.Monospace)
                Text(state.originalTotal.formattedAmount(), fontSize = 14.sp, fontWeight = FontWeight.Bold, color = lab.fg)
            }
            Column(horizontalAlignment = Alignment.End) {
                Text("Adjusted", fontSize = 10.sp, color = lab.fgTertiary, fontFamily = FontFamily.Monospace)
                Text(
                    state.adjustedTotal.formattedAmount(),
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Bold,
                    color = if (state.hasExclusions) StatusRed else StatusGreen
                )
            }
        }

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
                }
                if (index < state.audits.lastIndex) {
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant, thickness = 1.dp)
                }
            }
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

        // Shop Closed / No Answer button
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 4.dp)
        ) {
            androidx.compose.material3.OutlinedButton(
                onClick = { state.orderId?.let { onShopClosed(it) } },
                enabled = !state.isSubmitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(44.dp),
                shape = MaterialTheme.shapes.medium,
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = StatusOrange
                )
            ) {
                Icon(
                    Icons.Filled.RemoveCircleOutline,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                    tint = StatusOrange
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    text = "Shop Closed / No Answer",
                    fontWeight = FontWeight.Medium,
                    fontSize = 14.sp
                )
            }
        }

        // Edge 32: Credit Delivery button
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 4.dp)
        ) {
            androidx.compose.material3.OutlinedButton(
                onClick = { state.orderId?.let { onCreditDelivery(it) } },
                enabled = !state.isSubmitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(44.dp),
                shape = MaterialTheme.shapes.medium,
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = StatusBlue
                )
            ) {
                Icon(
                    Icons.Filled.CreditCard,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                    tint = StatusBlue
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    text = "Deliver on Credit",
                    fontWeight = FontWeight.Medium,
                    fontSize = 14.sp
                )
            }
        }

        // Edge 33: Report Missing Items button
        if (state.hasExclusions) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 4.dp)
            ) {
                androidx.compose.material3.OutlinedButton(
                    onClick = { state.orderId?.let { onReportMissing(it) } },
                    enabled = !state.isSubmitting,
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(44.dp),
                    shape = MaterialTheme.shapes.medium,
                    colors = ButtonDefaults.outlinedButtonColors(
                        contentColor = StatusRed
                    )
                ) {
                    Icon(
                        Icons.Filled.RemoveCircleOutline,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                        tint = StatusRed
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(
                        text = "Report Missing Items",
                        fontWeight = FontWeight.Medium,
                        fontSize = 14.sp
                    )
                }
            }
        }

        // Confirm button
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .background(MaterialTheme.colorScheme.surfaceContainerLow)
                .padding(16.dp)
        ) {
            Button(
                onClick = { viewModel.confirmOffload() },
                enabled = !state.isSubmitting,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
                shape = MaterialTheme.shapes.medium,
                colors = ButtonDefaults.buttonColors(containerColor = StatusGreen)
            ) {
                if (state.isSubmitting) {
                    CircularProgressIndicator(
                        color = MaterialTheme.colorScheme.onPrimary,
                        modifier = Modifier.size(20.dp),
                        strokeWidth = 2.dp
                    )
                } else {
                    Text(
                        text = if (state.hasExclusions) "Amend & Confirm Offload" else "Confirm Offload",
                        fontWeight = FontWeight.Bold,
                        fontSize = 15.sp
                    )
                }
            }
        }
    }
}
