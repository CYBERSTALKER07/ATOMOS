package com.pegasusx.supplier.ui.screens.inventory

import androidx.compose.runtime.Composable
import com.pegasusx.supplier.ui.screens.migration.MigrationCenterScreen

@Composable
fun InventoryImportScreen(
    onBack: () -> Unit,
) {
    MigrationCenterScreen(
        onBack = onBack,
        showRedirectBanner = true,
    )
}
