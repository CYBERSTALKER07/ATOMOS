package com.pegasusx.retailer.data.local

import androidx.room.Database
import androidx.room.migration.Migration
import androidx.room.RoomDatabase
import androidx.sqlite.db.SupportSQLiteDatabase

@Database(
    entities = [
        PendingOrderEntity::class,
        CatalogEntity::class,
        PredictionEntity::class
    ], 
    version = 3, 
    exportSchema = false
)
abstract class AppDatabase : RoomDatabase() {
    abstract fun pendingOrderDao(): PendingOrderDao
    abstract fun catalogDao(): CatalogDao
    abstract fun predictionDao(): PredictionDao

    companion object {
        val MIGRATION_1_2: Migration = object : Migration(1, 2) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL(
                    "ALTER TABLE pending_orders ADD COLUMN endpoint TEXT NOT NULL DEFAULT '/v1/checkout/unified'"
                )
                db.execSQL(
                    "ALTER TABLE pending_orders ADD COLUMN method TEXT NOT NULL DEFAULT 'POST'"
                )
                db.execSQL(
                    "ALTER TABLE pending_orders ADD COLUMN idempotencyKey TEXT NOT NULL DEFAULT ''"
                )
                db.execSQL(
                    "ALTER TABLE pending_orders ADD COLUMN lastError TEXT"
                )
                db.execSQL(
                    "UPDATE pending_orders SET idempotencyKey = 'retailer-checkout-pending:' || id || ':' || createdAt WHERE idempotencyKey = ''"
                )
            }
        }
        
        val MIGRATION_2_3: Migration = object : Migration(2, 3) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL(
                    "CREATE TABLE IF NOT EXISTS `catalog_items` (`id` TEXT NOT NULL, `name` TEXT NOT NULL, `price` REAL NOT NULL, `category` TEXT NOT NULL, `stock` INTEGER NOT NULL, `imageUrl` TEXT, PRIMARY KEY(`id`))"
                )
                db.execSQL(
                    "CREATE TABLE IF NOT EXISTS `demand_predictions` (`itemId` TEXT NOT NULL, `predictedDemand` INTEGER NOT NULL, `confidence` REAL NOT NULL, `timestamp` INTEGER NOT NULL, PRIMARY KEY(`itemId`))"
                )
            }
        }
    }
}
