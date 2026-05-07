# Technical Patent Architecture: Черновик Формулы Изобретения Pegasus (RU)

Source Document: i18n/patent-claims-draft.ru.md
Generated At: 2026-05-07T14:16:57.462Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- Настоящий документ является расширенным черновиком формулы изобретения для последующей юридической финализации. Текст подготовлен в патентно-ориентированном стиле и привязан к подтвержденным техническим модулям реализации.
- 1. Формульные и алгоритмические источники: patent-algorithm-atlas.ru.md.
- 1. Стратегическая и архитектурная рамка: patent-packet.ru.md.

## System Architecture
- Implementation Anchor: apps/backend-go/dispatch/binpack.go
- Implementation Anchor: apps/ai-worker/optimizer/clarke_wright.go
- Implementation Anchor: apps/ai-worker/optimizer/two_opt.go
- Implementation Anchor: apps/backend-go/proximity/recommendation.go
- Implementation Anchor: apps/backend-go/proximity/engine.go
- Implementation Anchor: apps/backend-go/factory/look_ahead.go
- Implementation Anchor: apps/backend-go/factory/supply_lanes.go
- Implementation Anchor: apps/backend-go/factory/network_optimizer.go
- Implementation Anchor: apps/backend-go/outbox/emit.go
- Implementation Anchor: apps/backend-go/outbox/relay.go
- Implementation Anchor: apps/backend-go/payment/gateway_client.go
- Implementation Anchor: apps/backend-go/payment/refund.go
- Implementation Anchor: apps/backend-go/idempotency/middleware.go
- Implementation Anchor: apps/admin-portal/lib/auth.ts
- Implementation Anchor: apps/admin-portal/lib/api/offlineQueue.ts
- Implementation Anchor: apps/admin-portal/lib/usePolling.ts
- Implementation Anchor: apps/driver-app-android/.../Haversine.kt
- Implementation Anchor: apps/driverappios/.../Utilities/Haversine.swift

## Feature Set
1. Раздел A. Независимый Способ И Зависимые Пункты
2. Раздел B. Независимая Система И Зависимые Пункты
3. Раздел C. Независимый Машиночитаемый Носитель И Зависимые Пункты
4. Раздел D. Варианты Выполнения Для Автономной Логистики
5. Раздел E. Технический Эффект (Для Сопроводительного Обоснования)
6. Раздел F. Карта Привязки К Реализации
7. Раздел G. Дополнительные Зависимые Пункты (Full Spectrum Extraction)
8. Раздел H. Альтернативные Варианты Выполнения (Future Embodiment Claims)

## Algorithmic and Logical Flow
- No algorithm or workflow section detected.

## Mathematical Formulations
- saving(i,j) = d(depot,i) + d(depot,j) - d(i,j).
- Пункт 21. Система по пункту 20, в которой целевой запас вычисляют как max(safety_level, ceil(future_demand * 1.15)).
- Пункт 23. Система по пункту 19, в которой обновление параметров линии поставки выполняют EMA-сглаживанием с коэффициентом alpha = 0.2.
- Пункт 27. Система по пункту 19, в которой скорость повторной попытки взаимодействия с внешним платежным шлюзом задают экспоненциальной функцией задержки 500 * 2^(attempt - 1).
- P_{total,minor}=P_{amount}\cdot100,
- P_{platform}=\left\lfloor\frac{P_{total,minor}\cdot feeBP}{10000}\right\rfloor,
- P_{supplier}=P_{total,minor}-P_{platform}.
- SettlementRate=\frac{TotalRevenue-CashPending}{TotalRevenue}.
- Пункт 56. Способ по пункту 4 или пункту 11, в котором передача телеметрической точки разрешена при выполнении хотя бы одного порога: $\Delta t>15$ секунд, $\Delta d>20$ метров или $\Delta\psi>15^\circ$.
- isFrozen(entityType,entityId)=1\iff now<t_{exp}.
- t_{exp}=t_{acq}+\max(ttl_{event},300\text{s}).
- P_{reroute}=1-\exp\left(-\frac{\Delta t_{delay}}{T_{buffer}}\right).

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- No explicit constraint block detected.

## Claims-Oriented Technical Elements
1. Feature family coverage includes Раздел A. Независимый Способ И Зависимые Пункты; Раздел B. Независимая Система И Зависимые Пункты; Раздел C. Независимый Машиночитаемый Носитель И Зависимые Пункты; Раздел D. Варианты Выполнения Для Автономной Логистики; Раздел E. Технический Эффект (Для Сопроводительного Обоснования); Раздел F. Карта Привязки К Реализации.
2. Mathematical or scoring expressions are explicitly used for optimization or estimation.
