package com.pegasus.payload.data.local

import androidx.room.Entity
import androidx.room.PrimaryKey

@Entity(tableName = "queued_actions")
data class QueuedActionEntity(
    @PrimaryKey
    val id: String,
    val endpoint: String,
    val method: String,
    val bodyJson: String?,
    val timestamp: Long,
    /** Capture-time GPS — replay on flush (§8.8). */
    val capturedLat: Double? = null,
    val capturedLng: Double? = null,
    val capturedAtMs: Long? = null,
    val attemptCount: Int = 0,
    val lastError: String = "",
    val status: String = "PENDING",
)
