package com.pegasus.barcode

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.Build
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.ui.platform.LocalContext

/**
 * Listens for common DataWedge / Zebra scan intents.
 *
 * Profile tips (device fleet): create a DataWedge profile for the warehouse app package,
 * enable Intent output with action `com.symbol.datawedge.api.RESULT_ACTION`,
 * delivery Broadcast, and key `com.symbol.datawedge.data_string`.
 */
@Composable
fun DataWedgeBarcodeEffect(onBarcode: (String) -> Unit) {
    val context = LocalContext.current
    DisposableEffect(onBarcode) {
        val receiver = object : BroadcastReceiver() {
            override fun onReceive(ctx: Context?, intent: Intent?) {
                if (intent == null) return
                val code = intent.getStringExtra("com.symbol.datawedge.data_string")
                    ?: intent.getStringExtra("com.motorolasolutions.emdk.datawedge.data_string")
                    ?: intent.getStringExtra("barcode_string")
                    ?: intent.getStringExtra("data")
                val trimmed = code?.trim().orEmpty()
                if (trimmed.isNotEmpty()) onBarcode(trimmed)
            }
        }
        val filter = IntentFilter().apply {
            addAction("com.symbol.datawedge.api.RESULT_ACTION")
            addAction("com.symbol.datawedge.datawedge.RESULT_ACTION")
            addCategory(Intent.CATEGORY_DEFAULT)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            context.registerReceiver(receiver, filter, Context.RECEIVER_EXPORTED)
        } else {
            @Suppress("UnspecifiedRegisterReceiverFlag")
            context.registerReceiver(receiver, filter)
        }
        onDispose {
            runCatching { context.unregisterReceiver(receiver) }
        }
    }
}
