package com.pegasusx.retailer.data.local

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "pending_pos_sales")
data class PendingPosSaleEntity(
    @PrimaryKey val id: String,
    val clientSaleId: String,
    val clientReceipt: String,
    val sessionId: String,
    val payloadJson: String,
    val idempotencyKey: String,
    val createdAt: Long = System.currentTimeMillis(),
    val retryCount: Int = 0,
    val status: String = "PENDING",
    val lastError: String? = null,
    val serverSaleId: String? = null,
    val serverReceiptNumber: String? = null,
)
