package com.pegasus.payload.di

import android.content.Context
import androidx.room.Room
import com.pegasus.payload.data.local.MIGRATION_1_2
import com.pegasus.payload.data.local.PayloadDatabase
import com.pegasus.payload.data.local.QueuedActionDao
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Singleton

@Module
@InstallIn(SingletonComponent::class)
object DatabaseModule {
    @Provides
    @Singleton
    fun providePayloadDatabase(@ApplicationContext context: Context): PayloadDatabase {
        return Room.databaseBuilder(
            context,
            PayloadDatabase::class.java,
            "payload_db"
        )
            .addMigrations(MIGRATION_1_2)
            .build()
    }

    @Provides
    fun provideQueuedActionDao(database: PayloadDatabase): QueuedActionDao {
        return database.queuedActionDao()
    }
}
