package com.pegasus.driver.util

import android.content.Context
import android.content.Intent
import android.net.Uri

object MapNavigation {
    fun openDeliveryLocation(context: Context, latitude: Double?, longitude: Double?, label: String? = null) {
        val lat = latitude ?: return
        val lng = longitude ?: return
        val title = Uri.encode(label?.takeIf { it.isNotBlank() } ?: "Delivery")
        val uri = Uri.parse("geo:$lat,$lng?q=$lat,$lng($title)")
        val intent = Intent(Intent.ACTION_VIEW, uri).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        context.startActivity(intent)
    }
}
