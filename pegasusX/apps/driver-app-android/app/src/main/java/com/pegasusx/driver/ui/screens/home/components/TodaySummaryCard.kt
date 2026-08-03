package com.pegasusx.driver.ui.screens.home.components

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.ui.components.DriverTodayKpiCard
import com.pegasusx.driver.ui.theme.formattedAmount
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Locale

@Composable
fun TodaySummaryCard(orders: List<Order>) {
    val pending = orders.count {
        it.state != OrderState.COMPLETED && it.state != OrderState.CANCELLED
    }
    val completed = orders.count { it.state == OrderState.COMPLETED }
    val revenue = orders
        .filter { it.state == OrderState.COMPLETED }
        .sumOf { it.totalAmount }

    val todayDate = remember {
        SimpleDateFormat("dd MMM yyyy", Locale.getDefault())
            .format(Calendar.getInstance().time)
            .uppercase()
    }

    DriverTodayKpiCard(
        dateLabel = todayDate,
        pending = pending,
        completed = completed,
        revenueLabel = if (revenue > 0) revenue.formattedAmount() else "—",
        pendingIcon = Icons.Default.Schedule,
        completedIcon = Icons.Default.CheckCircle,
        revenueIcon = Icons.Default.DirectionsCar,
    )
}
