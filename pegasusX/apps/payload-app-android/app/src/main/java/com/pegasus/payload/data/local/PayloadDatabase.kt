package com.pegasus.payload.data.local

import androidx.room.Database
import androidx.room.RoomDatabase

@Database(
    entities = [QueuedActionEntity::class],
    version = 2,
    exportSchema = false
)
abstract class PayloadDatabase : RoomDatabase() {
    abstract fun queuedActionDao(): QueuedActionDao
}
