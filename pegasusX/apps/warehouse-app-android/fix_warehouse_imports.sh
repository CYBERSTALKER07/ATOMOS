#!/bin/bash
sed -i '' 's/import androidx.compose.material3.\*/import androidx.compose.material3.*\nimport androidx.compose.material3.CardDefaults\nimport androidx.compose.material3.ButtonDefaults/g' app/src/main/java/com/pegasusx/warehouse/ui/screens/orders/OrderDetailScreen.kt
