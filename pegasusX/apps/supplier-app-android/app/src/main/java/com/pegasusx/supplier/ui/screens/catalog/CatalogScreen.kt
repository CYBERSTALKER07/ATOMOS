package com.pegasusx.supplier.ui.screens.catalog

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.pegasus.design.RealtimeRefreshEffect
import com.pegasus.design.showFullScreenLoading
import com.pegasus.barcode.EanBarcode
import com.pegasusx.supplier.data.model.CatalogCategory
import com.pegasusx.supplier.data.model.CatalogProduct
import com.pegasusx.supplier.data.model.CatalogProductCreateRequest
import com.pegasusx.supplier.data.model.CatalogProductUpdateRequest
import com.pegasusx.supplier.data.remote.CatalogImageUploader
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CatalogScreen(
    api: SupplierApi,
    realtimeSignals: SupplierRealtimeSignals,
    onOpenProduct: (String) -> Unit = {},
) {
    val context = LocalContext.current
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var products by remember { mutableStateOf(emptyList<CatalogProduct>()) }
    var categories by remember { mutableStateOf(emptyList<CatalogCategory>()) }
    var currency by remember { mutableStateOf("UZS") }
    val draftVU = remember { mutableStateMapOf<String, String>() }
    val draftBarcode = remember { mutableStateMapOf<String, String>() }
    val draftUnitsPerCase = remember { mutableStateMapOf<String, String>() }
    val draftSaleUnit = remember { mutableStateMapOf<String, String>() }
    var savingId by remember { mutableStateOf<String?>(null) }
    var showCreate by remember { mutableStateOf(false) }
    var creating by remember { mutableStateOf(false) }
    var createName by remember { mutableStateOf("") }
    var createPrice by remember { mutableStateOf("") }
    var createVu by remember { mutableStateOf("1") }
    var createBarcode by remember { mutableStateOf("") }
    var createSaleUnit by remember { mutableStateOf("UNIT") }
    var createUnitsPerCase by remember { mutableStateOf("") }
    var createCategoryId by remember { mutableStateOf("") }
    var createImageUri by remember { mutableStateOf<Uri?>(null) }
    var createImageLabel by remember { mutableStateOf<String?>(null) }
    var createError by remember { mutableStateOf<String?>(null) }
    var categoryMenuExpanded by remember { mutableStateOf(false) }
    var saleUnitMenuExpanded by remember { mutableStateOf(false) }
    var imageEditTargetId by remember { mutableStateOf<String?>(null) }
    var imageSavingId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getIntegerInstance(Locale.US) }

    val createImagePicker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        createImageUri = uri
        createImageLabel = uri?.lastPathSegment
    }

    val editImagePicker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        val targetId = imageEditTargetId
        imageEditTargetId = null
        if (uri == null || targetId == null) return@rememberLauncherForActivityResult
        val product = products.find { it.productId == targetId } ?: return@rememberLauncherForActivityResult
        scope.launch {
            imageSavingId = product.productId
            error = null
            try {
                val ext = CatalogImageUploader.fileExtension(uri.lastPathSegment)
                val imageUrl = CatalogImageUploader.uploadTicketImage(api, context, uri, ext)
                    .getOrElse { throw it }
                val resp = api.updateCatalogProduct(
                    product.productId,
                    CatalogProductUpdateRequest(
                        name = product.name,
                        priceMinor = product.priceMinor,
                        currency = product.currency,
                        unit = product.unit,
                        saleUnit = product.saleUnit,
                        unitsPerCase = product.unitsPerCase,
                        unitVolumeVu = product.unitVolumeVu,
                        imageUrl = imageUrl,
                        barcode = product.barcode,
                        isActive = product.isActive,
                        version = product.version,
                    ),
                )
                if (!resp.isSuccessful) {
                    error = "Image update failed (${resp.code()})"
                    return@launch
                }
                val updated = resp.body() ?: return@launch
                products = products.map { row ->
                    if (row.productId == updated.productId) updated else row
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                imageSavingId = null
            }
        }
    }

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) {
                loading = true
                error = null
            }
            try {
                val profileResp = api.getProfile()
                if (profileResp.isSuccessful) {
                    currency = profileResp.body()?.currency?.ifBlank { "UZS" } ?: "UZS"
                }
                val resp = api.listCatalogProducts()
                if (resp.isSuccessful) {
                    products = resp.body().orEmpty()
                    draftVU.clear()
                    draftBarcode.clear()
                    draftUnitsPerCase.clear()
                    draftSaleUnit.clear()
                } else if (!silent) {
                    error = "Failed (${resp.code()})"
                    products = emptyList()
                }
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    fun loadCategories() {
        scope.launch {
            try {
                val profile = api.getProfile().body()
                val resp = api.listCatalogCategories(profile?.supplierId)
                categories = if (resp.isSuccessful) resp.body().orEmpty() else emptyList()
                if (createCategoryId.isBlank()) {
                    createCategoryId = categories.firstOrNull()?.categoryId.orEmpty()
                }
            } catch (_: Exception) {
                categories = emptyList()
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    RealtimeRefreshEffect(
        refreshTick = realtimeSignals.refreshTick,
        reconnectTick = realtimeSignals.reconnectTick,
        onRefresh = { load(silent = it) },
    )

    fun saveUnitVolume(product: CatalogProduct) {
        val raw = draftVU[product.productId] ?: product.unitVolumeVu.toString()
        val parsed = raw.toDoubleOrNull()
        if (parsed == null || parsed <= 0.0) {
            error = "Unit VU must be a positive number."
            return
        }
        val saleUnit = draftSaleUnit[product.productId] ?: product.saleUnit
        val unitsPerCaseRaw = draftUnitsPerCase[product.productId]
            ?: product.unitsPerCase?.toString().orEmpty()
        val unitsPerCase = if (saleUnit == "CASE") {
            val parsedUnits = unitsPerCaseRaw.toLongOrNull()
            if (parsedUnits == null || parsedUnits <= 0L) {
                error = "Units per case must be a positive integer when selling by case."
                return
            }
            parsedUnits
        } else {
            null
        }
        scope.launch {
            savingId = product.productId
            error = null
            try {
                val barcodeRaw = draftBarcode[product.productId] ?: product.barcode.orEmpty()
                val barcode = barcodeRaw.trim().takeIf { it.isNotEmpty() }?.let {
                    EanBarcode.normalize(it) ?: run {
                        error = "Invalid EAN/GTIN barcode."
                        savingId = null
                        return@launch
                    }
                }
                val resp = api.updateCatalogProduct(
                    product.productId,
                    CatalogProductUpdateRequest(
                        name = product.name,
                        priceMinor = product.priceMinor,
                        currency = product.currency,
                        unit = product.unit,
                        unitVolumeVu = parsed,
                        unitsPerCase = unitsPerCase,
                        saleUnit = saleUnit,
                        barcode = barcode,
                        isActive = product.isActive,
                        version = product.version,
                    ),
                )
                if (!resp.isSuccessful) {
                    error = "Save failed (${resp.code()})"
                    return@launch
                }
                val updated = resp.body() ?: return@launch
                products = products.map { row ->
                    if (row.productId == updated.productId) updated else row
                }
                draftVU.remove(product.productId)
                draftBarcode.remove(product.productId)
                draftUnitsPerCase.remove(product.productId)
                draftSaleUnit.remove(product.productId)
            } catch (e: Exception) {
                error = e.message
            } finally {
                savingId = null
            }
        }
    }

    fun createProduct() {
        val name = createName.trim()
        val priceMinor = createPrice.toLongOrNull()
        val unitVolume = createVu.toDoubleOrNull()
        if (name.isBlank() || createCategoryId.isBlank()) {
            createError = "Name and category are required."
            return
        }
        if (priceMinor == null || priceMinor < 0) {
            createError = "Price must be a non-negative integer."
            return
        }
        if (unitVolume == null || unitVolume <= 0.0) {
            createError = "Unit VU must be positive."
            return
        }
        val barcode = createBarcode.trim().takeIf { it.isNotEmpty() }?.let {
            EanBarcode.normalize(it) ?: run {
                createError = "Invalid EAN/GTIN barcode."
                return
            }
        }
        val unitsPerCase = if (createSaleUnit == "CASE") {
            val parsedUnits = createUnitsPerCase.toLongOrNull()
            if (parsedUnits == null || parsedUnits <= 0L) {
                createError = "Units per case must be a positive integer when selling by case."
                return
            }
            parsedUnits
        } else {
            null
        }
        scope.launch {
            creating = true
            createError = null
            try {
                var imageUrl: String? = null
                createImageUri?.let { uri ->
                    val ext = CatalogImageUploader.fileExtension(createImageLabel)
                    imageUrl = CatalogImageUploader.uploadTicketImage(api, context, uri, ext)
                        .getOrElse { throw it }
                }
                val resp = api.createCatalogProduct(
                    CatalogProductCreateRequest(
                        categoryId = createCategoryId,
                        name = name,
                        priceMinor = priceMinor,
                        currency = currency,
                        unitVolumeVu = unitVolume,
                        unitsPerCase = unitsPerCase,
                        saleUnit = createSaleUnit,
                        imageUrl = imageUrl,
                        barcode = barcode,
                    ),
                )
                if (!resp.isSuccessful) {
                    createError = "Create failed (${resp.code()})"
                    return@launch
                }
                showCreate = false
                createName = ""
                createPrice = ""
                createVu = "1"
                createBarcode = ""
                createSaleUnit = "UNIT"
                createUnitsPerCase = ""
                createImageUri = null
                createImageLabel = null
                load()
            } catch (e: Exception) {
                createError = e.message
            } finally {
                creating = false
            }
        }
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("Catalog") }) },
        floatingActionButton = {
            FloatingActionButton(
                onClick = {
                    showCreate = true
                    createError = null
                    loadCategories()
                },
            ) {
                Icon(Icons.Default.Add, contentDescription = "Add product")
            }
        },
    ) { padding ->
        Box(Modifier.padding(padding).fillMaxSize()) {
            when {
                showFullScreenLoading(loading, products.isNotEmpty()) -> PegasusLoadingState("Loading catalog…", "Product VU")
                error != null && products.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Catalog unavailable",
                    body = error!!,
                )
                products.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No products",
                    body = "Tap + to create a product and set unit volume.",
                )
                else -> LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 340.dp),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    if (error != null) {
                        item {
                            Text(
                                error!!,
                                color = androidx.compose.material3.MaterialTheme.colorScheme.error,
                                modifier = Modifier.padding(bottom = PegasusSpacing.sm),
                            )
                        }
                    }
                    items(products, key = { it.productId }) { product ->
                        val vuValue = draftVU[product.productId] ?: product.unitVolumeVu.toString()
                        val barcodeValue = draftBarcode[product.productId]
                            ?: product.barcode.orEmpty()
                        val saleUnit = draftSaleUnit[product.productId] ?: product.saleUnit
                        val unitsPerCaseValue = draftUnitsPerCase[product.productId]
                            ?: product.unitsPerCase?.toString().orEmpty()
                        val vuDirty = vuValue != product.unitVolumeVu.toString()
                        val barcodeDirty = barcodeValue != product.barcode.orEmpty()
                        val saleUnitDirty = saleUnit != product.saleUnit
                        val unitsPerCaseDirty = unitsPerCaseValue != product.unitsPerCase?.toString().orEmpty()
                        val dirty = vuDirty || barcodeDirty || saleUnitDirty || unitsPerCaseDirty
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(
                                modifier = Modifier.padding(PegasusSpacing.lg),
                                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                            ) {
                                Text(product.name, style = androidx.compose.material3.MaterialTheme.typography.titleMedium)
                                TextButton(onClick = { onOpenProduct(product.productId) }) {
                                    Text("View details")
                                }
                                Text(
                                    "${fmt.format(product.priceMinor)} ${product.currency} · ${product.unit}",
                                    style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
                                    color = androidx.compose.material3.MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                Text(
                                    "Sale: ${saleUnit.lowercase()}${if (saleUnit == "CASE" && unitsPerCaseValue.isNotBlank()) " ($unitsPerCaseValue/case)" else ""}",
                                    style = androidx.compose.material3.MaterialTheme.typography.labelSmall,
                                    color = androidx.compose.material3.MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                if (!product.imageUrl.isNullOrBlank()) {
                                    Text(
                                        "Image attached",
                                        style = androidx.compose.material3.MaterialTheme.typography.labelSmall,
                                        color = androidx.compose.material3.MaterialTheme.colorScheme.tertiary,
                                    )
                                }
                                TextButton(
                                    onClick = {
                                        imageEditTargetId = product.productId
                                        editImagePicker.launch("image/*")
                                    },
                                    enabled = imageSavingId != product.productId,
                                ) {
                                    Text(
                                        when {
                                            imageSavingId == product.productId -> "Uploading…"
                                            product.imageUrl.isNullOrBlank() -> "Add image"
                                            else -> "Change image"
                                        },
                                    )
                                }
                                CatalogBarcodeField(
                                    value = barcodeValue,
                                    onValueChange = { draftBarcode[product.productId] = it },
                                    enabled = savingId != product.productId,
                                )
                                var rowSaleUnitExpanded by remember(product.productId) { mutableStateOf(false) }
                                ExposedDropdownMenuBox(
                                    expanded = rowSaleUnitExpanded,
                                    onExpandedChange = { rowSaleUnitExpanded = it },
                                ) {
                                    OutlinedTextField(
                                        value = if (saleUnit == "CASE") "Case" else "Unit",
                                        onValueChange = {},
                                        readOnly = true,
                                        label = { Text("Sale unit") },
                                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = rowSaleUnitExpanded) },
                                        modifier = Modifier
                                            .menuAnchor()
                                            .fillMaxWidth(),
                                    )
                                    ExposedDropdownMenu(
                                        expanded = rowSaleUnitExpanded,
                                        onDismissRequest = { rowSaleUnitExpanded = false },
                                    ) {
                                        listOf("UNIT" to "Unit", "CASE" to "Case").forEach { (value, label) ->
                                            DropdownMenuItem(
                                                text = { Text(label) },
                                                onClick = {
                                                    draftSaleUnit[product.productId] = value
                                                    rowSaleUnitExpanded = false
                                                },
                                            )
                                        }
                                    }
                                }
                                if (saleUnit == "CASE") {
                                    OutlinedTextField(
                                        value = unitsPerCaseValue,
                                        onValueChange = { draftUnitsPerCase[product.productId] = it },
                                        label = { Text("Units per case") },
                                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                                        singleLine = true,
                                        modifier = Modifier.fillMaxWidth(),
                                    )
                                }
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                                ) {
                                    OutlinedTextField(
                                        value = vuValue,
                                        onValueChange = { draftVU[product.productId] = it },
                                        label = { Text("Unit VU") },
                                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                                        singleLine = true,
                                        modifier = Modifier.widthIn(min = 120.dp, max = 160.dp),
                                    )
                                    Button(
                                        onClick = { saveUnitVolume(product) },
                                        enabled = dirty && savingId != product.productId,
                                    ) {
                                        Text(if (savingId == product.productId) "…" else "Save")
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showCreate) {
        AlertDialog(
            onDismissRequest = { if (!creating) showCreate = false },
            title = { Text("Add product") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    OutlinedTextField(
                        value = createName,
                        onValueChange = { createName = it },
                        label = { Text("Name") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    ExposedDropdownMenuBox(
                        expanded = categoryMenuExpanded,
                        onExpandedChange = { categoryMenuExpanded = it },
                    ) {
                        OutlinedTextField(
                            value = categories.find { it.categoryId == createCategoryId }?.name ?: "Select category",
                            onValueChange = {},
                            readOnly = true,
                            label = { Text("Category") },
                            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = categoryMenuExpanded) },
                            modifier = Modifier
                                .menuAnchor()
                                .fillMaxWidth(),
                        )
                        ExposedDropdownMenu(
                            expanded = categoryMenuExpanded,
                            onDismissRequest = { categoryMenuExpanded = false },
                        ) {
                            categories.forEach { category ->
                                DropdownMenuItem(
                                    text = { Text(category.name) },
                                    onClick = {
                                        createCategoryId = category.categoryId
                                        categoryMenuExpanded = false
                                    },
                                )
                            }
                        }
                    }
                    OutlinedTextField(
                        value = createPrice,
                        onValueChange = { createPrice = it },
                        label = { Text("Price ($currency, minor)") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = createVu,
                        onValueChange = { createVu = it },
                        label = { Text("Unit VU") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                    ExposedDropdownMenuBox(
                        expanded = saleUnitMenuExpanded,
                        onExpandedChange = { saleUnitMenuExpanded = it },
                    ) {
                        OutlinedTextField(
                            value = if (createSaleUnit == "CASE") "Case" else "Unit",
                            onValueChange = {},
                            readOnly = true,
                            label = { Text("Sale unit") },
                            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = saleUnitMenuExpanded) },
                            modifier = Modifier
                                .menuAnchor()
                                .fillMaxWidth(),
                        )
                        ExposedDropdownMenu(
                            expanded = saleUnitMenuExpanded,
                            onDismissRequest = { saleUnitMenuExpanded = false },
                        ) {
                            listOf("UNIT" to "Unit", "CASE" to "Case").forEach { (value, label) ->
                                DropdownMenuItem(
                                    text = { Text(label) },
                                    onClick = {
                                        createSaleUnit = value
                                        saleUnitMenuExpanded = false
                                    },
                                )
                            }
                        }
                    }
                    if (createSaleUnit == "CASE") {
                        OutlinedTextField(
                            value = createUnitsPerCase,
                            onValueChange = { createUnitsPerCase = it },
                            label = { Text("Units per case") },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                    CatalogBarcodeField(
                        value = createBarcode,
                        onValueChange = { createBarcode = it },
                        enabled = !creating,
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        TextButton(onClick = { createImagePicker.launch("image/*") }) {
                            Text(if (createImageLabel != null) "Image selected" else "Add image")
                        }
                    }
                    createError?.let {
                        Text(it, color = androidx.compose.material3.MaterialTheme.colorScheme.error)
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = { createProduct() },
                    enabled = !creating && categories.isNotEmpty(),
                ) {
                    Text(if (creating) "Creating…" else "Create")
                }
            },
            dismissButton = {
                TextButton(onClick = { showCreate = false }, enabled = !creating) {
                    Text("Cancel")
                }
            },
        )
    }
}
