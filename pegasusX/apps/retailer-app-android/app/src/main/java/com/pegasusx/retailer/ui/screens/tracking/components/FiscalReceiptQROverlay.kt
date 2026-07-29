package com.pegasusx.retailer.ui.screens.tracking.components

import android.graphics.Bitmap
import android.graphics.Color as AndroidColor
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter
import com.pegasusx.retailer.data.model.TrackingOrder

@Composable
fun FiscalReceiptQROverlay(
    order: TrackingOrder?,
    onDismiss: () -> Unit,
) {
    if (order == null || order.fiscalQr.isBlank()) return
    Dialog(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(24.dp))
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                "Fiscal receipt",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
            )
            Text(
                order.fiscalReceiptLabel,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
            val bitmap = remember(order.fiscalQr) {
                runCatching {
                    val size = 512
                    val matrix = QRCodeWriter().encode(order.fiscalQr, BarcodeFormat.QR_CODE, size, size)
                    Bitmap.createBitmap(size, size, Bitmap.Config.ARGB_8888).also { bmp ->
                        for (x in 0 until size) {
                            for (y in 0 until size) {
                                bmp.setPixel(
                                    x,
                                    y,
                                    if (matrix[x, y]) AndroidColor.BLACK else AndroidColor.WHITE,
                                )
                            }
                        }
                    }
                }.getOrNull()
            }
            if (bitmap != null) {
                Image(
                    bitmap = bitmap.asImageBitmap(),
                    contentDescription = "Fiscal QR",
                    modifier = Modifier.size(220.dp),
                )
            } else {
                Text(
                    order.fiscalQr,
                    style = MaterialTheme.typography.bodySmall,
                    textAlign = TextAlign.Center,
                )
            }
            if (order.latestFiscalReceiptId.isNotBlank()) {
                Text(
                    "ID · ${order.latestFiscalReceiptId}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            TextButton(onClick = onDismiss) {
                Text("Close")
            }
        }
    }
}
