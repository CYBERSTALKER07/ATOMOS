package com.pegasusx.supplier.ui.screens.catalog

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
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
                else -> CatalogList(
                    products = products,
                    draftVU = draftVU,
                    draftBarcode = draftBarcode,
                    draftUnitsPerCase = draftUnitsPerCase,
                    draftSaleUnit = draftSaleUnit,
                    savingId = savingId,
                    imageSavingId = imageSavingId,
                    error = error,
                    onSaveUnitVolume = ::saveUnitVolume,
                    onOpenProduct = onOpenProduct,
                    onChangeImage = { 
                        imageEditTargetId = it
                        editImagePicker.launch("image/*") 
                    }
                )
            }
        }
    }

    if (showCreate) {
        CreateProductDialog(
            categories = categories,
            currency = currency,
            creating = creating,
            createName = createName,
            onNameChange = { createName = it },
            createPrice = createPrice,
            onPriceChange = { createPrice = it },
            createVu = createVu,
            onVuChange = { createVu = it },
            createBarcode = createBarcode,
            onBarcodeChange = { createBarcode = it },
            createSaleUnit = createSaleUnit,
            onSaleUnitChange = { createSaleUnit = it },
            createUnitsPerCase = createUnitsPerCase,
            onUnitsPerCaseChange = { createUnitsPerCase = it },
            createCategoryId = createCategoryId,
            onCategoryIdChange = { createCategoryId = it },
            createImageLabel = createImageLabel,
            createImagePicker = createImagePicker,
            createError = createError,
            categoryMenuExpanded = categoryMenuExpanded,
            onCategoryMenuExpandedChange = { categoryMenuExpanded = it },
            saleUnitMenuExpanded = saleUnitMenuExpanded,
            onSaleUnitMenuExpandedChange = { saleUnitMenuExpanded = it },
            onCreateProduct = ::createProduct,
            onDismiss = { showCreate = false }
        )
    }
}
