# Technical Patent Architecture: Атлас Алгоритмов Pegasus (RU)

Source Document: i18n/patent-algorithm-atlas.ru.md
Generated At: 2026-05-07T14:16:57.461Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- Этот документ фиксирует алгоритмическое ядро экосистемы Pegasus в форме, удобной для патентной проработки: формулы, пороги, инварианты и связь с конкретными модулями реализации.
- 1. Сначала читать как карту уже реализованной машинной логики.
- 1. Затем читать как источник формулировок для формулы-притязаний и функциональных зависимых пунктов.

## System Architecture
- Implementation Anchor: apps/backend-go/dispatch/binpack.go
- Implementation Anchor: apps/backend-go/dispatch/cluster.go
- Implementation Anchor: apps/backend-go/dispatch/split.go
- Implementation Anchor: apps/backend-go/dispatch/service.go
- Implementation Anchor: apps/backend-go/routing/optimizer.go
- Implementation Anchor: apps/ai-worker/optimizer/clarke_wright.go
- Implementation Anchor: apps/ai-worker/optimizer/two_opt.go
- Implementation Anchor: apps/ai-worker/optimizer/solver.go
- Implementation Anchor: apps/backend-go/proximity/h3.go
- Implementation Anchor: apps/backend-go/proximity/serving_warehouse.go
- Implementation Anchor: apps/backend-go/proximity/warehouse_resolver.go
- Implementation Anchor: apps/backend-go/proximity/recommendation.go
- Implementation Anchor: apps/backend-go/proximity/engine.go
- Implementation Anchor: apps/backend-go/factory/pull_matrix.go
- Implementation Anchor: apps/backend-go/factory/look_ahead.go
- Implementation Anchor: apps/backend-go/factory/predictive_push.go
- Implementation Anchor: apps/backend-go/factory/network_optimizer.go
- Implementation Anchor: apps/backend-go/factory/supply_lanes.go
- Implementation Anchor: apps/backend-go/factory/replenishment_lock.go
- Implementation Anchor: apps/backend-go/payment/refund.go
- Implementation Anchor: apps/backend-go/payment/gateway_client.go
- Implementation Anchor: apps/backend-go/payment/global_pay.go
- Implementation Anchor: apps/backend-go/analytics/financials.go
- Implementation Anchor: apps/backend-go/idempotency/middleware.go
- Implementation Anchor: apps/backend-go/cache/priority.go
- Implementation Anchor: apps/backend-go/cache/ratelimit.go
- Implementation Anchor: apps/backend-go/cache/circuitbreaker.go
- Implementation Anchor: apps/backend-go/cache/pubsub.go
- Implementation Anchor: apps/backend-go/telemetry/metrics.go
- Implementation Anchor: apps/admin-portal/lib/auth.ts

## Feature Set
1. 1. Диспетчеризация И Маршрутизация
2. 2. Геопространственная Логика И Геозона
3. 3. Пополнение Запаса, Прогноз И Межузловой Трансфер
4. 4. Платежный Контур, Рефанд И Финансовая Непротиворечивость
5. 5. Устойчивость: Backpressure, Rate Limit, Circuit, Idempotency
6. 6. Мобильные И Терминальные Алгоритмы
7. 7. Инфраструктурные Алгоритмы И Данные
8. 8. Матрица Притязаний По Формульному Ядру
9. 9. Отметка О Границах Доказательности
10. 10. Full Spectrum Extraction (Code-Verified, May 2026)
11. 10.1 Геопространственная H3-Формула (текущая реализация)
12. 10.2 Capacity Bin-Packing / Max-VU (текущая реализация)
13. 10.3 Telemetry / Dead Reckoning (текущая реализация)
14. 10.4 AI Demand / Replenishment Формулы (текущая реализация)
15. 10.5 Финансовый Контур: Split + Refund + Settlement (текущая реализация)
16. 10.7 Payload Sealing Idempotency (текущая реализация)
17. 10.8 Матрица Статуса Формул (IP Hygiene)

## Algorithmic and Logical Flow
- No algorithm or workflow section detected.

## Mathematical Formulations
- 1. Буфер укладки транспорта: effective_capacity = nominal_capacity * 0.95 (TetrisBuffer).
- 1. Сбережение Кларка-Райта: saving(i,j) = d(depot,i) + d(depot,j) - d(i,j), далее добавляется приоритетный буст для восстановительных заказов.
- 1. Ограничение маршрута по числу остановок: max_stops_per_route = 25.
- 1. Пространственный индекс H3: resolution = 7, строковое представление ячейки длиной 15 hex-символов.
- 1. Буфер safety stock: 15%, целевой уровень: target_stock = max(safety_level, ceil(future_demand * 1.15)).
- 1. Оценка числа рейсов класса C: convoy_count = ceil(total_vu / 400.0).
- 1. EMA сглаживание lane-метрик: ema_next = alpha * sample + (1 - alpha) * ema_prev, alpha = 0.2.
- 1. BALANCED сетевой режим: score = transit * 0.5 + freight * 0.0003 + carbon * 0.2.
- 1. Ретрай внешнего шлюза: delay_ms = 500 * 2^(attempt - 1) с ограничением числа попыток.
- C_{grid}(h_w,h_r)=\max(0.01,\delta(h_w,h_r)\cdot \ell_{r=7})\cdot \Pi(\rho)
- \Pi(\rho)=
- где $\delta(h_w,h_r)$ — H3 ring-distance, $\ell_{r=7}$ — длина ребра ячейки (константа резолюции 7), $\rho$ — load percent склада.
- C'(h_w,h_r)=\delta(h_w,h_r)\cdot \lambda + \sum_{i=1}^{\delta}\Omega(h_i)
- V_{eff}=0.95\cdot V_{nom}
- 2. Объем заказа (Kahan compensated sum):
- V_{order}=\sum_i q_i\cdot v_i
- N_{chunks}=\left\lceil\frac{V_{order}}{V_{fleet,max}^{eff}}\right\rceil
- v_k=\min(V_{remaining},V_{fleet,max}^{eff})
- S_{ij}=d(0,i)+d(0,j)-d(i,j)+\pi_i+\pi_j
- T_{send}=1 \iff (\Delta t>15\text{s})\;\lor\;(\Delta d>20\text{m})\;\lor\;(\Delta\psi>15^\circ)

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- **Ключевые файлы**
- 1. pegasus/apps/backend-go/cache/priority.go
- 1. pegasus/apps/backend-go/cache/ratelimit.go
- 1. pegasus/apps/backend-go/cache/circuitbreaker.go
- 1. pegasus/apps/backend-go/cache/pubsub.go
- 1. pegasus/apps/backend-go/telemetry/metrics.go
- 1. pegasus/apps/admin-portal/lib/auth.ts
- 1. pegasus/apps/admin-portal/lib/api/offlineQueue.ts
- 1. pegasus/apps/admin-portal/lib/usePolling.ts
- 1. pegasus/apps/admin-portal/lib/useSyncHub.ts
- **Формулы И Правила**
- 1. Приоритетное shed-управление делит запросы по классам критичности и динамически режет низшие классы под давлением.
- 1. Redis token bucket исполняется атомарно (Lua), исключая гонки параллельных воркеров.
- 1. Circuit breaker: closed -> open -> half-open с учетом окна ошибок и cooldown.
- 1. WebSocket pubsub fail-open: локальная доставка сохраняется даже при ошибке межподовой публикации.
- 1. В админ-портале mutating-запросы получают Idempotency-Key автоматически, сетевые ошибки отправляют мутацию в offline queue.
- 1. В offline queue элементы старше 24 часов удаляются при дренировании.
- 1. Backpressure сигнал от backend передается через X-Backpressure-Interval и снижает polling cadence на клиенте.
- **Патентная ценность**
- 1. Единый контур деградации качества сервиса предотвращает каскадный отказ без потери юридически значимых событий заказа и оплаты.
- 1. Клиентский ключ идемпотентности:
- $$
- K=\text{"payload-"}+action+\text{"-"}+entityId

## Claims-Oriented Technical Elements
1. Feature family coverage includes 1. Диспетчеризация И Маршрутизация; 2. Геопространственная Логика И Геозона; 3. Пополнение Запаса, Прогноз И Межузловой Трансфер; 4. Платежный Контур, Рефанд И Финансовая Непротиворечивость; 5. Устойчивость: Backpressure, Rate Limit, Circuit, Idempotency; 6. Мобильные И Терминальные Алгоритмы.
2. Mathematical or scoring expressions are explicitly used for optimization or estimation.
3. Integrity constraints include **Ключевые файлы**; 1. pegasus/apps/backend-go/cache/priority.go; 1. pegasus/apps/backend-go/cache/ratelimit.go; 1. pegasus/apps/backend-go/cache/circuitbreaker.go.
