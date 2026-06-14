package com.pegasusx.supplier.ui.portal

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.Payments
import androidx.compose.material.icons.filled.PersonAdd
import androidx.compose.material.icons.filled.Settings
import androidx.compose.ui.graphics.vector.ImageVector

/** Portal-only supplier surfaces — native apps hand off to supplier-portal. */
enum class SupplierPortalFeature(
    val routeKey: String,
    val title: String,
    val subtitle: String,
    val portalPath: String,
    val icon: ImageVector,
) {
    REGISTER(
        routeKey = "register",
        title = "Register supplier",
        subtitle = "Create a new supplier account",
        portalPath = "/auth/register",
        icon = Icons.Default.PersonAdd,
    ),
    BUSINESS_SETUP(
        routeKey = "business_setup",
        title = "Business setup",
        subtitle = "Tax ID, address, and company profile",
        portalPath = "/setup/business",
        icon = Icons.Default.Settings,
    ),
    CHARGEBACKS(
        routeKey = "chargebacks",
        title = "Chargebacks",
        subtitle = "Payment disputes and reversals",
        portalPath = "/payments",
        icon = Icons.Default.Payments,
    ),
    PAYMENT_BYPASS(
        routeKey = "payment_bypass",
        title = "Payment bypass",
        subtitle = "High-consequence operator actions",
        portalPath = "/operations",
        icon = Icons.Default.CreditCard,
    ),
    ;

    val handoffMessage: String
        get() = when (this) {
            REGISTER -> "Supplier registration runs on the supplier web portal."
            BUSINESS_SETUP -> "Business profile setup is completed on the supplier web portal."
            CHARGEBACKS -> "Chargeback review and treasury actions are managed on the supplier portal."
            PAYMENT_BYPASS -> "Payment bypass is available in Operations on native and web."
        }

    companion object {
        fun fromRouteKey(key: String): SupplierPortalFeature? =
            entries.firstOrNull { it.routeKey == key }
    }
}
