package com.pegasusx.supplier.ui.screens.catalog

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.CatalogProduct
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CatalogDetailScreen(
    productId: String,
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var product by remember { mutableStateOf<CatalogProduct?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(productId) {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getCatalogProduct(productId)
                if (resp.isSuccessful) {
                    product = resp.body()
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Product detail") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        Box(Modifier.padding(padding)) {
            when {
                loading -> PegasusLoadingState("Loading product…", productId.take(12))
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Product unavailable",
                    body = error!!,
                )
                product == null -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "Not found",
                    body = "No product for this ID.",
                )
                else -> {
                    val p = product!!
                    Column(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(PegasusSpacing.lg),
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    ) {
                        Text(p.name, style = MaterialTheme.typography.headlineSmall)
                        Text(stringResource(R.string.mobile_supplier_ui_id_productid, p.productId), style = MaterialTheme.typography.bodySmall)
                        Text(stringResource(R.string.mobile_supplier_ui_category_categoryid, p.categoryId), style = MaterialTheme.typography.bodyMedium)
                        Text(
                            stringResource(R.string.mobile_supplier_ui_price_formatminoramount, formatMinorAmount(p.priceMinor, p.currency)),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        Text(stringResource(R.string.mobile_supplier_ui_unit_unit_vu_unitvolumevu, p.unit, p.unitVolumeVu), style = MaterialTheme.typography.bodyMedium)
                        p.barcode?.let { Text(stringResource(R.string.mobile_supplier_ui_barcode_it, it), style = MaterialTheme.typography.bodyMedium) }
                        Text(
                            if (p.isActive) "Active" else "Inactive",
                            style = MaterialTheme.typography.labelLarge,
                            color = if (p.isActive) MaterialTheme.colorScheme.primary
                            else MaterialTheme.colorScheme.error,
                        )
                    }
                }
            }
        }
    }
}
