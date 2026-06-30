package com.pegasus.retailer.data.local

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "cart_cache")
data class CartCacheEntity(
    @PrimaryKey val retailerId: String,
    val itemsJson: String,
    val updatedAt: Long = System.currentTimeMillis(),
)
