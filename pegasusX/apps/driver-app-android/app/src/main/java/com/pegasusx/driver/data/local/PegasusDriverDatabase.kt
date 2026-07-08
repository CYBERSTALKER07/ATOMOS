package com.pegasusx.driver.data.local

import androidx.room.Database
import androidx.room.RoomDatabase
import com.pegasusx.driver.data.model.OrderEntity
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.data.model.RouteManifestEntity

import com.pegasusx.driver.data.model.TelemetryLocationEntity

@Database(
    entities = [OrderEntity::class, RouteManifestEntity::class, PendingMutationEntity::class, TelemetryLocationEntity::class],
    version = 4,
    exportSchema = false
)
abstract class PegasusDriverDatabase : RoomDatabase() {
    abstract fun orderDao(): OrderDao
    abstract fun routeManifestDao(): RouteManifestDao
    abstract fun pendingMutationDao(): PendingMutationDao
    abstract fun telemetryDao(): TelemetryDao
}
