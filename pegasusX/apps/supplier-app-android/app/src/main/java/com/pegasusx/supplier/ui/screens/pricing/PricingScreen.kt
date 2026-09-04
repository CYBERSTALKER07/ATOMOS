package com.pegasusx.supplier.ui.screens.pricing

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import com.pegasusx.supplier.data.model.CatalogProduct
import com.pegasusx.supplier.data.model.CatalogProductUpdateRequest
import com.pegasusx.supplier.data.model.SupplierPromotion
import com.pegasusx.supplier.data.model.SupplierPromotionUpsertRequest
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PricingScreen(api: SupplierApi, onBack: () -> Unit) {
    var products by remember { mutableStateOf<List<CatalogProduct>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selectedProduct by remember { mutableStateOf<CatalogProduct?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.listCatalogProducts()
                products = if (resp.isSuccessful) resp.body().orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Pricing") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading pricing…", "Catalog products", Modifier.padding(padding))
            error != null && products.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Pricing unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            products.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No products to price",
                body = "Add products in Catalog first. They will appear here for list and sale pricing.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(products, key = { it.productId }) { product ->
                    SupplierOpsListCard(
                        headline = product.name,
                        supporting = formatMinorAmount(product.priceMinor, product.currency),
                        status = if (product.isActive) "ACTIVE" else "INACTIVE",
                        onClick = { selectedProduct = product },
                    )
                }
            }
        }
    }

    selectedProduct?.let { product ->
        ProductPricingDialog(
            api = api,
            product = product,
            onDismiss = { selectedProduct = null },
            onSaved = {
                selectedProduct = null
                load()
            },
        )
    }
}

@Composable
private fun ProductPricingDialog(
    api: SupplierApi,
    product: CatalogProduct,
    onDismiss: () -> Unit,
    onSaved: () -> Unit,
) {
    var priceMajor by remember(product.productId) {
        mutableStateOf(String.format("%.2f", product.priceMinor / 100.0))
    }
    var saleEnabled by remember { mutableStateOf(false) }
    var saleDiscountBps by remember { mutableStateOf("") }
    var activeSale by remember { mutableStateOf<SupplierPromotion?>(null) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(product.productId) {
        val promoResp = api.getPromotions()
        if (promoResp.isSuccessful) {
            val promo = promoResp.body()?.promotions.orEmpty().firstOrNull {
                it.isActive && it.scopeType == "PRODUCT" && it.scopeProductId == product.productId
            }
            activeSale = promo
            if (promo != null) {
                saleEnabled = true
                saleDiscountBps = promo.discountBps.toString()
            }
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(product.name) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(
                    value = priceMajor,
                    onValueChange = { priceMajor = it },
                    label = { Text(stringResource(R.string.mobile_supplier_ui_list_price_currency, product.currency)) },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    singleLine = true,
                )
                Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
                    Text("On sale", modifier = Modifier.weight(1f))
                    Switch(checked = saleEnabled, onCheckedChange = { saleEnabled = it })
                }
                if (saleEnabled) {
                    OutlinedTextField(
                        value = saleDiscountBps,
                        onValueChange = { saleDiscountBps = it },
                        label = { Text("Discount (bps)") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        singleLine = true,
                    )
                }
                error?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
            }
        },
        confirmButton = {
            TextButton(
                enabled = !saving,
                onClick = {
                    scope.launch {
                        saving = true
                        error = null
                        val clean = priceMajor.replace(',', '.')
                        val parts = clean.split(".")
                        val major = parts.getOrNull(0)?.toLongOrNull() ?: 0L
                        val minor = parts.getOrNull(1)?.padEnd(2, '0')?.substring(0, 2)?.toLongOrNull() ?: 0L
                        val priceMinor: Long? = if (major > 0 || minor > 0) (major * 100) + minor else null
                        if (priceMinor == null || priceMinor < 0) {
                            error = "Enter a valid list price."
                            saving = false
                            return@launch
                        }
                        try {
                            val updateResp = api.updateCatalogProduct(
                                product.productId,
                                CatalogProductUpdateRequest(
                                    name = product.name,
                                    priceMinor = priceMinor,
                                    currency = product.currency,
                                    unit = product.unit,
                                    saleUnit = product.saleUnit,
                                    unitsPerCase = product.unitsPerCase,
                                    unitVolumeVu = product.unitVolumeVu,
                                    imageUrl = product.imageUrl,
                                    barcode = product.barcode,
                                    isActive = product.isActive,
                                    version = product.version,
                                ),
                            )
                            if (!updateResp.isSuccessful) error("update_failed_${updateResp.code()}")

                            if (saleEnabled) {
                                val bps = saleDiscountBps.toLongOrNull() ?: 0
                                if (bps <= 0) error("Sale discount must be greater than zero.")
                                val request = SupplierPromotionUpsertRequest(
                                    name = "Sale · ${product.name}",
                                    description = "Product sale pricing",
                                    discountBps = bps,
                                    scopeType = "PRODUCT",
                                    scopeProductId = product.productId,
                                    retailerScope = "ALL",
                                )
                                val promoResp = if (activeSale != null) {
                                    api.updatePromotion(activeSale!!.promotionId, request)
                                } else {
                                    api.createPromotion(request)
                                }
                                if (!promoResp.isSuccessful) error("promotion_failed_${promoResp.code()}")
                            } else if (activeSale != null) {
                                api.deactivatePromotion(activeSale!!.promotionId)
                            }
                            onSaved()
                        } catch (e: Exception) {
                            error = e.message
                        } finally {
                            saving = false
                        }
                    }
                },
            ) { Text(if (saving) "Saving…" else "Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
