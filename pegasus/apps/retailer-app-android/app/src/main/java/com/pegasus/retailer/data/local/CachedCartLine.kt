package com.pegasus.retailer.data.local

import kotlinx.serialization.Serializable

@Serializable
data class CachedCartLine(
    val id: String,
    val skuId: String,
    val supplierId: String,
    val quantity: Int,
    val unitPrice: Double,
    val productName: String,
    val variantId: String,
    val variantSize: String = "Standard",
    val variantPack: String = "Per unit",
)
