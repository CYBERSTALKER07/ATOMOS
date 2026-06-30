package com.pegasus.retailer.data.local

import androidx.room.Database
import androidx.room.migration.Migration
import androidx.room.RoomDatabase
import androidx.sqlite.db.SupportSQLiteDatabase

@Database(entities = [PendingOrderEntity::class, CartCacheEntity::class], version = 3, exportSchema = false)
abstract class AppDatabase : RoomDatabase() {
    abstract fun pendingOrderDao(): PendingOrderDao
    abstract fun cartCacheDao(): CartCacheDao

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
                    """
                    CREATE TABLE IF NOT EXISTS cart_cache (
                        retailerId TEXT NOT NULL PRIMARY KEY,
                        itemsJson TEXT NOT NULL,
                        updatedAt INTEGER NOT NULL
                    )
                    """.trimIndent()
                )
            }
        }
    }
}
