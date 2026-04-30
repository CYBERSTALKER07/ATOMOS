package com.thelab.retailer.data.local

import androidx.room.Database
import androidx.room.RoomDatabase

@Database(entities = [PendingOrderEntity::class], version = 1, exportSchema = false)
abstract class AppDatabase : RoomDatabase() {
    abstract fun pendingOrderDao(): PendingOrderDao
}
