package com.pegasusx.driver.ui.screens.offload.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.RemoveCircleOutline
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.ui.theme.StatusBlue
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusOrange
import com.pegasusx.driver.ui.theme.StatusRed

@Composable
fun OffloadActionFooter(
    isSubmitting: Boolean,
    isUploadingPhoto: Boolean,
    hasRejections: Boolean,
    orderId: String?,
    onShopClosed: (String) -> Unit,
    onCreditDelivery: () -> Unit,
    onReportMissing: (String) -> Unit,
    onConfirm: () -> Unit
) {
    // Shop Closed / No Answer button
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 4.dp)
    ) {
        androidx.compose.material3.OutlinedButton(
            onClick = { orderId?.let { onShopClosed(it) } },
            enabled = !isSubmitting,
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
                text = stringResource(R.string.mobile_driver_ui_shop_closed_no_answer),
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
            onClick = onCreditDelivery,
            enabled = !isSubmitting,
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
                text = stringResource(R.string.mobile_driver_ui_deliver_on_credit),
                fontWeight = FontWeight.Medium,
                fontSize = 14.sp
            )
        }
    }

    // Edge 33: Report Missing Items button
    if (hasRejections) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 4.dp)
        ) {
            androidx.compose.material3.OutlinedButton(
                onClick = { orderId?.let { onReportMissing(it) } },
                enabled = !isSubmitting,
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
                    text = stringResource(R.string.mobile_driver_ui_report_missing_items),
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
            onClick = onConfirm,
            enabled = !isSubmitting && !isUploadingPhoto,
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
            shape = MaterialTheme.shapes.medium,
            colors = ButtonDefaults.buttonColors(containerColor = StatusGreen)
        ) {
            if (isSubmitting) {
                CircularProgressIndicator(
                    color = MaterialTheme.colorScheme.onPrimary,
                    modifier = Modifier.size(20.dp),
                    strokeWidth = 2.dp
                )
            } else {
                Text(
                    text = if (hasRejections) "Amend & Confirm Offload" else "Confirm Offload",
                    fontWeight = FontWeight.Bold,
                    fontSize = 15.sp
                )
            }
        }
    }
}
