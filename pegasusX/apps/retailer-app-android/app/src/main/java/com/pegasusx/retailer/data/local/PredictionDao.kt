package com.pegasusx.retailer.data.local

import androidx.room.*
import kotlinx.coroutines.flow.Flow

@Entity(tableName = "demand_predictions")
data class PredictionEntity(
    @PrimaryKey val itemId: String,
    val predictedDemand: Int,
    val confidence: Double,
    val timestamp: Long
)

@Dao
interface PredictionDao {
    @Query("SELECT * FROM demand_predictions")
    fun getAll(): Flow<List<PredictionEntity>>

    @Query("SELECT * FROM demand_predictions WHERE itemId = :itemId")
    suspend fun getByItemId(itemId: String): PredictionEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(predictions: List<PredictionEntity>)

    @Query("DELETE FROM demand_predictions")
    suspend fun clearAll()
}
