package com.pegasusx.retailer.ui.components

import androidx.compose.ui.res.stringResource

import android.graphics.Bitmap
import android.os.Build
import android.view.WindowManager
import androidx.compose.foundation.Image
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.QrCode2
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.compose.ui.window.DialogWindowProvider
import com.pegasusx.retailer.data.model.Order
import com.google.zxing.BarcodeFormat
import com.google.zxing.qrcode.QRCodeWriter
import org.json.JSONObject

@Composable
fun QROverlay(
    visible: Boolean,
    order: Order?,
    onDismiss: () -> Unit,
) {
    // QR is only available after dispatch (EN_ROUTE state). Immediately dismiss for other states.
    if (visible && order != null && order.status.hasDeliveryToken) {
        Dialog(
            onDismissRequest = onDismiss,
            properties = DialogProperties(
                usePlatformDefaultWidth = false,
                decorFitsSystemWindows = false,
            ),
        ) {
            // Make dialog window background transparent and add blur behind
            val view = LocalView.current
            SideEffect {
                val w = (view.parent as? DialogWindowProvider)?.window
                w?.let {
                    it.setDimAmount(0.1f)
                    it.setBackgroundDrawableResource(android.R.color.transparent)
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                        it.addFlags(WindowManager.LayoutParams.FLAG_BLUR_BEHIND)
                        it.attributes = it.attributes.apply {
                            blurBehindRadius = 30
                        }
                    }
                }
            }

            // Frosted scrim
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.White.copy(alpha = 0.75f))
                    .clickable(
                        interactionSource = remember { MutableInteractionSource() },
                        indication = null,
                    ) { onDismiss() },
                contentAlignment = Alignment.Center,
            ) {
                Column(
                    modifier = Modifier
                        .padding(40.dp)
                        .clip(SoftSquircleShape)
                        .background(MaterialTheme.colorScheme.surface)
                        .clickable(
                            interactionSource = remember { MutableInteractionSource() },
                            indication = null,
                        ) { /* consume cash */ }
                        .padding(32.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                ) {
                    Text(
                        stringResource(R.string.mobile_retailer_ui_order_takelast, order.id.takeLast(3)),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Spacer(modifier = Modifier.height(24.dp))

                    DeliveryQrCode(order)

                    Spacer(modifier = Modifier.height(20.dp))
                    Text(
                        "Show to driver for\ndelivery confirmation",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                        textAlign = TextAlign.Center,
                    )

                    Spacer(modifier = Modifier.height(24.dp))
                    Text(
                        "Dismiss",
                        style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier
                            .clip(PillShape)
                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f))
                            .clickable { onDismiss() }
                            .padding(horizontal = 32.dp, vertical = 12.dp),
                    )
                }
            }
        }
    }
}

@Composable
private fun DeliveryQrCode(order: Order) {
    val payload = buildDeliveryQrPayload(order)
    val bitmap = remember(payload) { payload?.let { generateQrBitmap(it, 720, 720) } }

    if (bitmap != null) {
        Image(
            bitmap = bitmap.asImageBitmap(),
            contentDescription = stringResource(R.string.mobile_retailer_ui_qr_code),
            modifier = Modifier.size(180.dp),
        )
    } else {
        Icon(
            Icons.Outlined.QrCode2,
            contentDescription = stringResource(R.string.mobile_retailer_ui_qr_code),
            modifier = Modifier.size(180.dp),
            tint = MaterialTheme.colorScheme.onSurface,
        )
    }
}

private fun buildDeliveryQrPayload(order: Order): String? {
    val token = order.qrCode?.takeIf { it.isNotBlank() } ?: return null
    return JSONObject(
        mapOf(
            "order_id" to order.id,
            "token" to token,
        ),
    ).toString()
}

private fun generateQrBitmap(data: String, width: Int, height: Int): Bitmap? {
    return runCatching {
        val matrix = QRCodeWriter().encode(data, BarcodeFormat.QR_CODE, width, height)
        val pixels = IntArray(width * height)
        for (y in 0 until height) {
            for (x in 0 until width) {
                pixels[y * width + x] = if (matrix[x, y]) android.graphics.Color.BLACK else android.graphics.Color.WHITE
            }
        }
        Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888).apply {
            setPixels(pixels, 0, width, 0, 0, width, height)
        }
    }.getOrNull()
}
