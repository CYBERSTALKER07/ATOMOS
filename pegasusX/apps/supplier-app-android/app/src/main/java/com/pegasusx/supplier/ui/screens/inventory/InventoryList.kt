package com.pegasusx.supplier.ui.screens.inventory

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.ListItem
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.data.model.InventoryItem

@Composable
fun InventoryList(
    items: List<InventoryItem>,
    onShowAdjust: (String) -> Unit,
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        items(items, key = { it.sku }) { item ->
            ListItem(
                headlineContent = { Text(item.productName) },
                supportingContent = { Text(stringResource(R.string.mobile_supplier_ui_sku_sku_qty_quantity, item.sku, item.quantity)) },
                modifier = Modifier.clickable { onShowAdjust(item.sku) },
            )
        }
    }
}
