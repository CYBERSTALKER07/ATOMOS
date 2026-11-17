package com.pegasusx.retailer.ui.screens.tracking.components

import androidx.compose.ui.res.stringResource

import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Color as AndroidColor
import android.net.Uri
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter
import com.pegasusx.retailer.BuildConfig
import com.pegasusx.retailer.data.model.TrackingOrder
import com.pegasusx.retailer.R

@Composable
fun FiscalReceiptQROverlay(
    order: TrackingOrder?,
    onDismiss: () -> Unit,
) {
    if (order == null || order.fiscalQr.isBlank()) return
    val context = LocalContext.current
    val openUrl: (String) -> Unit = { url ->
        runCatching {
            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
        }
    }
    val htmlUrl = remember(order.fiscalQr, order.latestFiscalReceiptId) {
        when {
            order.fiscalQr.contains("format=") -> order.fiscalQr
            order.fiscalQr.isNotBlank() -> {
                val sep = if (order.fiscalQr.contains("?")) "&" else "?"
                "${order.fiscalQr}${sep}format=html"
            }
            order.latestFiscalReceiptId.isNotBlank() ->
                "${BuildConfig.BASE_URL}v1/platform/receipts/${order.latestFiscalReceiptId}?format=html"
            else -> ""
        }
    }
    val pdfUrl = remember(order.latestFiscalReceiptId, order.orderId) {
        when {
            order.latestFiscalReceiptId.isNotBlank() ->
                "${BuildConfig.BASE_URL}v1/platform/receipts/${order.latestFiscalReceiptId}?format=pdf"
            order.orderId.isNotBlank() ->
                "${BuildConfig.BASE_URL}v1/retailer/orders/${order.orderId}/receipt?format=pdf"
            else -> ""
        }
    }
    Dialog(onDismissRequest = onDismiss) {
        Column(
            modifier = Modifier
                .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(24.dp))
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                "Pegasus receipt",
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
                    contentDescription = stringResource(R.string.mobile_retailer_ui_fiscal_qr),
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
                    stringResource(R.string.mobile_retailer_ui_id_latestfiscalreceiptid, order.latestFiscalReceiptId),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                if (htmlUrl.isNotBlank()) {
                    TextButton(onClick = { openUrl(htmlUrl) }) {
                        Text("Open receipt")
                    }
                }
                if (pdfUrl.isNotBlank()) {
                    TextButton(onClick = { openUrl(pdfUrl) }) {
                        Text("PDF")
                    }
                }
                TextButton(onClick = onDismiss) {
                    Text("Close")
                }
            }
        }
    }
}
