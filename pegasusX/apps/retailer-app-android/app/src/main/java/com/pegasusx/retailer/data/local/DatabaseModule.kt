package com.pegasusx.retailer.data.local

import android.content.Context
import androidx.room.Room
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
    fun provideDatabase(@ApplicationContext context: Context): AppDatabase =
        Room.databaseBuilder(context, AppDatabase::class.java, "retailer_db")
            .addMigrations(AppDatabase.MIGRATION_1_2, AppDatabase.MIGRATION_2_3)
            .build()

    @Provides
    fun providePendingOrderDao(db: AppDatabase): PendingOrderDao = db.pendingOrderDao()

    @Provides
    fun provideCatalogDao(db: AppDatabase): CatalogDao = db.catalogDao()

    @Provides
    fun providePredictionDao(db: AppDatabase): PredictionDao = db.predictionDao()
}
