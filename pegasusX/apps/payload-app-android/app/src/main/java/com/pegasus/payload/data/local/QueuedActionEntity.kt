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
    val timestamp: Long
)
