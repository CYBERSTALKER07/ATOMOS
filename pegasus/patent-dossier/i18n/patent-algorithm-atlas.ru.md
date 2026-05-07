# Атлас Алгоритмов Pegasus (RU)

**Назначение**
Этот документ фиксирует алгоритмическое ядро экосистемы Pegasus в форме, удобной для патентной проработки: формулы, пороги, инварианты и связь с конкретными модулями реализации.

**Режим Чтения**
1. Сначала читать как карту уже реализованной машинной логики.
1. Затем читать как источник формулировок для формулы-притязаний и функциональных зависимых пунктов.
1. Отдельно отмечать, где логика уже продуктивна, а где указана как будущая автономная эмбодиментация.

## 1. Диспетчеризация И Маршрутизация

**Ключевые файлы**
1. pegasus/apps/backend-go/dispatch/binpack.go
1. pegasus/apps/backend-go/dispatch/cluster.go
1. pegasus/apps/backend-go/dispatch/split.go
1. pegasus/apps/backend-go/dispatch/service.go
1. pegasus/apps/backend-go/routing/optimizer.go
1. pegasus/apps/ai-worker/optimizer/clarke_wright.go
1. pegasus/apps/ai-worker/optimizer/two_opt.go
1. pegasus/apps/ai-worker/optimizer/solver.go

**Формулы И Правила**
1. Буфер укладки транспорта: effective_capacity = nominal_capacity * 0.95 (TetrisBuffer).
1. Объем позиции заказа считается с компенсированной суммой Кэхэна для снижения ошибки накопления.
1. Сбережение Кларка-Райта: saving(i,j) = d(depot,i) + d(depot,j) - d(i,j), далее добавляется приоритетный буст для восстановительных заказов.
1. Локальная оптимизация 2-opt применяется ограниченно по итерациям (дефолт 200).
1. Ограничение маршрута по числу остановок: max_stops_per_route = 25.
1. ETA-приближение использует среднюю скорость по умолчанию 30.0 км/ч и сервисное время 5 минут на точку при отсутствии точной матрицы.

**Патентная ценность**
1. Сочетание бинпакинга с гео-кластеризацией и bounded local search формирует воспроизводимый машинный порядок исполнения, устойчивый к изменению UI.

## 2. Геопространственная Логика И Геозона

**Ключевые файлы**
1. pegasus/apps/backend-go/proximity/h3.go
1. pegasus/apps/backend-go/proximity/serving_warehouse.go
1. pegasus/apps/backend-go/proximity/warehouse_resolver.go
1. pegasus/apps/backend-go/proximity/recommendation.go
1. pegasus/apps/backend-go/proximity/engine.go

**Формулы И Правила**
1. Пространственный индекс H3: resolution = 7, строковое представление ячейки длиной 15 hex-символов.
1. Нелинейный штраф нагрузки склада: до 70% используется мягкий режим, после 70% штраф усиливается квадратично (load^2-профиль).
1. Пороги автопереназначения: trigger при score > 0.9, целевая зона стабилизации при score < 0.7.
1. Геозона завершения заказа: контроль дистанции между текущей позицией исполнителя и точкой ритейлера с пороговым допуском (конфигурационный).

**Патентная ценность**
1. Ролевая логистика опирается не на сырые координаты, а на устойчивую клеточную геометрию и управляемые пороги риска.

## 3. Пополнение Запаса, Прогноз И Межузловой Трансфер

**Ключевые файлы**
1. pegasus/apps/backend-go/factory/pull_matrix.go
1. pegasus/apps/backend-go/factory/look_ahead.go
1. pegasus/apps/backend-go/factory/predictive_push.go
1. pegasus/apps/backend-go/factory/network_optimizer.go
1. pegasus/apps/backend-go/factory/supply_lanes.go
1. pegasus/apps/backend-go/factory/replenishment_lock.go

**Формулы И Правила**
1. Окно look-ahead: 7 дней.
1. Буфер safety stock: 15%, целевой уровень: target_stock = max(safety_level, ceil(future_demand * 1.15)).
1. Оценка числа рейсов класса C: convoy_count = ceil(total_vu / 400.0).
1. EMA сглаживание lane-метрик: ema_next = alpha * sample + (1 - alpha) * ema_prev, alpha = 0.2.
1. Порог обновления lane-параметров: минимум 15% изменения и не чаще одного раза в час.
1. BALANCED сетевой режим: score = transit * 0.5 + freight * 0.0003 + carbon * 0.2.
1. TTL lock пополнения: 10 минут, допускается приоритетное вытеснение более слабой блокировки.

**Патентная ценность**
1. Контур сочетает threshold-детекцию, прогнозный push и приоритетные блокировки, образуя предотвращающую дефицит саморегуляцию сети.

## 4. Платежный Контур, Рефанд И Финансовая Непротиворечивость

**Ключевые файлы**
1. pegasus/apps/backend-go/payment/refund.go
1. pegasus/apps/backend-go/payment/gateway_client.go
1. pegasus/apps/backend-go/payment/global_pay.go
1. pegasus/apps/backend-go/analytics/financials.go
1. pegasus/apps/backend-go/idempotency/middleware.go

**Формулы И Правила**
1. Расщепление рефанда выполняется в минорных единицах валюты по basis points, с обратной проводкой долей.
1. Ретрай внешнего шлюза: delay_ms = 500 * 2^(attempt - 1) с ограничением числа попыток.
1. Платежная сессия hosted checkout имеет таймаутный lifecycle sweep (просрочка и закрытие stale-сессий).
1. Settlement-rate рассчитывается как отношение успешно финализированных платежей к общему валидному объему за интервал.

**Патентная ценность**
1. Денежный результат логистического события не отделен от его operational state machine и формально устойчив к replay/timeout сбоям.

## 5. Устойчивость: Backpressure, Rate Limit, Circuit, Idempotency

**Ключевые файлы**
1. pegasus/apps/backend-go/cache/priority.go
1. pegasus/apps/backend-go/cache/ratelimit.go
1. pegasus/apps/backend-go/cache/circuitbreaker.go
1. pegasus/apps/backend-go/cache/pubsub.go
1. pegasus/apps/backend-go/telemetry/metrics.go
1. pegasus/apps/admin-portal/lib/auth.ts
1. pegasus/apps/admin-portal/lib/api/offlineQueue.ts
1. pegasus/apps/admin-portal/lib/usePolling.ts
1. pegasus/apps/admin-portal/lib/useSyncHub.ts

**Формулы И Правила**
1. Приоритетное shed-управление делит запросы по классам критичности и динамически режет низшие классы под давлением.
1. Redis token bucket исполняется атомарно (Lua), исключая гонки параллельных воркеров.
1. Circuit breaker: closed -> open -> half-open с учетом окна ошибок и cooldown.
1. WebSocket pubsub fail-open: локальная доставка сохраняется даже при ошибке межподовой публикации.
1. В админ-портале mutating-запросы получают Idempotency-Key автоматически, сетевые ошибки отправляют мутацию в offline queue.
1. В offline queue элементы старше 24 часов удаляются при дренировании.
1. Backpressure сигнал от backend передается через X-Backpressure-Interval и снижает polling cadence на клиенте.

**Патентная ценность**
1. Единый контур деградации качества сервиса предотвращает каскадный отказ без потери юридически значимых событий заказа и оплаты.

## 6. Мобильные И Терминальные Алгоритмы

**Подтвержденные реализации с формулой/порогом**
1. pegasus/apps/driver-app-android/.../Haversine.kt: расчет great-circle distance, радиус Земли 6_371_000 м.
1. pegasus/apps/driver-app-android/.../TelemetryService.kt: отправка позиции при отклонении дистанции > 20 м или курса > 15 градусов.
1. pegasus/apps/driverappios/.../Utilities/Haversine.swift: эквивалентный расчет дистанции для iOS.
1. pegasus/apps/payload-terminal/utils/manifest.ts: нормализация и суммирование позиций манифеста по единицам.

**Подтвержденные протоколы без фиксированного численного порога в текущем срезе**
1. RETAILER Android/iOS/Desktop: офлайн-согласование заказа, карточные статусы, live updates.
1. FACTORY Android/iOS/Portal: supply-request lifecycle, transfer/manifests coordination.
1. WAREHOUSE Android/iOS/Portal: dispatch-lock и supply-request websocket контур.
1. PAYLOAD iOS/Android/Terminal: синхронизация исключений манифеста и seal/progress событий.

**Патентная ценность**
1. Алгоритм является системным, а не экранным: разные клиенты реализуют единый машинный контракт при различной UI-оболочке.

## 7. Инфраструктурные Алгоритмы И Данные

**Ключевые файлы/поверхности**
1. pegasus/apps/backend-go/outbox/emit.go
1. pegasus/apps/backend-go/outbox/relay.go
1. pegasus/docker-compose.yml
1. pegasus/infra/terraform
1. pegasus/scripts/contract_guard_mcp.py
1. pegasus/scripts/architecture_guard_mcp.py
1. pegasus/scripts/design_system_guard_mcp.py
1. pegasus/scripts/production_safety_guard.py

**Правила**
1. Transactional outbox: запись состояния и события в одной Spanner RW-транзакции, последующая relay-публикация в Kafka.
1. Producer key привязывается к aggregate root id для сохранения партиционной причинности.
1. Cache invalidation публикуется через Redis Pub/Sub после успешного коммита.
1. Guard-скрипты блокируют опасный дрейф контракта и архитектурных границ.

**Патентная ценность**
1. Обеспечивается доказуемая причинно-следственная трасса от мутации до события и клиентского обновления.

## 8. Матрица Притязаний По Формульному Ядру

1. Формульная диспетчеризация: бинпакинг + savings + 2-opt + route constraints.
1. Формульная геозона и proximity-рерутинг: H3, пороги риска, нелинейный штраф нагрузки.
1. Формульное пополнение: look-ahead дефицит, safety buffer, EMA lane stabilization.
1. Формульный финконтур: basis-point распределение рефанда, backoff и settlement-реконсиляция.
1. Формульная отказоустойчивость: token bucket, backpressure shed, circuit transition, idempotent replay.

## 9. Отметка О Границах Доказательности

1. Численные константы в этом атласе взяты из текущей реализованной логики backend-go, ai-worker и подтвержденных клиентских модулей.
1. Для role-row экранов, где в текущем срезе подтвержден протокол, но не фиксированная численная константа, указана архитектурная связка без искусственной цифры.
1. Будущие автономные режимы вынесены в отдельный документ future-autonomous-vision.ru.md и не смешиваются с блоком implemented-now.
