package com.pegasusx.driver.data.local

import androidx.room.migration.Migration
import androidx.sqlite.db.SupportSQLiteDatabase

/** v5 → v6: capture-time lat/lng/at on pending_mutations (§8.8). */
val MIGRATION_5_6 = object : Migration(5, 6) {
    override fun migrate(db: SupportSQLiteDatabase) {
        db.execSQL("ALTER TABLE pending_mutations ADD COLUMN capturedLat REAL")
        db.execSQL("ALTER TABLE pending_mutations ADD COLUMN capturedLng REAL")
        db.execSQL("ALTER TABLE pending_mutations ADD COLUMN capturedAtMs INTEGER")
    }
}
