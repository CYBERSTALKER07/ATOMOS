package com.pegasus.payload.data.local

import androidx.room.migration.Migration
import androidx.sqlite.db.SupportSQLiteDatabase

val MIGRATION_1_2 = object : Migration(1, 2) {
    override fun migrate(db: SupportSQLiteDatabase) {
        db.execSQL("ALTER TABLE queued_actions ADD COLUMN capturedLat REAL")
        db.execSQL("ALTER TABLE queued_actions ADD COLUMN capturedLng REAL")
        db.execSQL("ALTER TABLE queued_actions ADD COLUMN capturedAtMs INTEGER")
        db.execSQL("ALTER TABLE queued_actions ADD COLUMN attemptCount INTEGER NOT NULL DEFAULT 0")
        db.execSQL("ALTER TABLE queued_actions ADD COLUMN lastError TEXT NOT NULL DEFAULT ''")
        db.execSQL("ALTER TABLE queued_actions ADD COLUMN status TEXT NOT NULL DEFAULT 'PENDING'")
    }
}
