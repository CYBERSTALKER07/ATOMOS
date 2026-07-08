package com.pegasusx.retailer.data.local

import androidx.room.*
import kotlinx.coroutines.flow.Flow

@Entity(tableName = "catalog_items")
data class CatalogEntity(
    @PrimaryKey val id: String,
    val name: String,
    val price: Double,
    val category: String,
    val stock: Int,
    val imageUrl: String?
)

@Dao
interface CatalogDao {
    @Query("SELECT * FROM catalog_items")
    fun getAll(): Flow<List<CatalogEntity>>

    @Query("SELECT * FROM catalog_items WHERE id = :id")
    suspend fun getById(id: String): CatalogEntity?

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(items: List<CatalogEntity>)

    @Query("DELETE FROM catalog_items")
    suspend fun clearAll()
}
