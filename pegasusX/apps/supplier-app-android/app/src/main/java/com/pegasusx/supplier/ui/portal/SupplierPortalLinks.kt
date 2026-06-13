package com.pegasusx.supplier.ui.portal

import android.net.Uri
import com.pegasusx.supplier.BuildConfig

object SupplierPortalLinks {
    fun url(feature: SupplierPortalFeature): String {
        val base = BuildConfig.PORTAL_BASE_URL.trimEnd('/')
        val path = feature.portalPath.trimStart('/')
        return if (path.isEmpty()) base else "$base/$path"
    }

    fun openUri(feature: SupplierPortalFeature): Uri = Uri.parse(url(feature))
}
