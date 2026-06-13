package com.pegasusx.warehouse.ui.portal

import android.net.Uri
import com.pegasusx.warehouse.BuildConfig

object WarehousePortalLinks {
    fun url(feature: WarehousePortalFeature): String {
        val base = BuildConfig.PORTAL_BASE_URL.trimEnd('/')
        val path = feature.portalPath.trimStart('/')
        return if (path.isEmpty()) base else "$base/$path"
    }

    fun openUri(feature: WarehousePortalFeature): Uri = Uri.parse(url(feature))
}
