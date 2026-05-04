package com.pegasus.retailer.data.local

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "pending_orders")
data class PendingOrderEntity(
    @PrimaryKey(autoGenerate = true) val id: Long = 0,
    val endpoint: String = "/v1/checkout/unified",
    val method: String = "POST",
    val payloadJson: String,
    val idempotencyKey: String,
    val createdAt: Long = System.currentTimeMillis(),
    val retryCount: Int = 0,
    val lastError: String? = null,
)
