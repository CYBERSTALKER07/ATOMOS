package com.thelab.driver.data.local

import androidx.room.Database
import androidx.room.RoomDatabase
import com.thelab.driver.data.model.OrderEntity
import com.thelab.driver.data.model.PendingMutationEntity
import com.thelab.driver.data.model.RouteManifestEntity

@Database(
    entities = [OrderEntity::class, RouteManifestEntity::class, PendingMutationEntity::class],
    version = 2,
    exportSchema = false
)
abstract class LabDriverDatabase : RoomDatabase() {
    abstract fun orderDao(): OrderDao
    abstract fun routeManifestDao(): RouteManifestDao
    abstract fun pendingMutationDao(): PendingMutationDao
}
