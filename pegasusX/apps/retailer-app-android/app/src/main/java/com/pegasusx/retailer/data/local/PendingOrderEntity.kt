package com.pegasusx.retailer.data.local

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
    val status: String = STATUS_PENDING,
) {
    companion object {
        const val STATUS_PENDING = "PENDING"
        const val STATUS_AWAITING_OWNER_APPROVAL = "AWAITING_OWNER_APPROVAL"
        const val STATUS_IN_PROGRESS = "IN_PROGRESS"
        const val STATUS_RATE_LIMITED = "RATE_LIMITED"
    }
}
