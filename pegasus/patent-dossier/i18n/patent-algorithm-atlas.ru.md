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

## 10. Full Spectrum Extraction (Code-Verified, May 2026)

**Назначение секции**
1. Зафиксировать математическое ядро в режиме "что реально исполняется сейчас".
1. Отдельно зафиксировать "future embodiment" формулы, которые уместны для патентного объема, но пока не закодированы в runtime.

### 10.1 Геопространственная H3-Формула (текущая реализация)

**Реализовано**
1. Для H3-grid scoring используется:

$$
C_{grid}(h_w,h_r)=\max(0.01,\delta(h_w,h_r)\cdot \ell_{r=7})\cdot \Pi(\rho)
$$

$$
\Pi(\rho)=
\begin{cases}
1+\rho, & \rho\le 0.70 \\
1+\rho^2, & \rho>0.70
\end{cases}
$$

где $\delta(h_w,h_r)$ — H3 ring-distance, $\ell_{r=7}$ — длина ребра ячейки (константа резолюции 7), $\rho$ — load percent склада.

2. Для lat/lng scoring используется Haversine distance с тем же нелинейным штрафом нагрузки.

**Future embodiment (не runtime)**
1. "Cellular Friction Index" модель допустима как расширение:

$$
C'(h_w,h_r)=\delta(h_w,h_r)\cdot \lambda + \sum_{i=1}^{\delta}\Omega(h_i)
$$

но в текущем коде суммирование $\Omega(h_i)$ по промежуточным ячейкам не выполняется.

### 10.2 Capacity Bin-Packing / Max-VU (текущая реализация)

**Реализовано**
1. Эффективная вместимость транспорта:

$$
V_{eff}=0.95\cdot V_{nom}
$$

2. Объем заказа (Kahan compensated sum):

$$
V_{order}=\sum_i q_i\cdot v_i
$$

с компенсированным накоплением для снижения float-error.

3. Для oversized заказа:

$$
N_{chunks}=\left\lceil\frac{V_{order}}{V_{fleet,max}^{eff}}\right\rceil
$$

$$
v_k=\min(V_{remaining},V_{fleet,max}^{eff})
$$

4. Сбережение Кларка-Райта с приоритетным бустом recovery-заказов:

$$
S_{ij}=d(0,i)+d(0,j)-d(i,j)+\pi_i+\pi_j
$$

**Важно по freeze lock**
1. В reassignment-ветке freeze lock реализован как hard-gate (конфликт), а не как бесконечный вес в objective-функции.

### 10.3 Telemetry / Dead Reckoning (текущая реализация)

**Реализовано (platform-specific)**
1. На driver-app-android отправка точки выполняется при условии:

$$
T_{send}=1 \iff (\Delta t>15\text{s})\;\lor\;(\Delta d>20\text{m})\;\lor\;(\Delta\psi>15^\circ)
$$

иначе $T_{send}=0$.

2. На driverappios отправка точки выполняется при условии минимального смещения:

$$
T_{send,iOS}=1 \iff \Delta d\ge 10\text{m}
$$

3. Auto-ARRIVED геозона (Android execution path):

$$
d_{hav}(P_{driver},P_{target})\le 100\text{m}
$$

**Future embodiment (не runtime)**
1. Ускорительная модель

$$
\vec{P}_{pred}=\vec{P}_{last}+\vec{V}_{last}\Delta t+\frac{1}{2}\vec{A}_{avg}(\Delta t)^2
$$

в текущей мобильной логике не применяется.

### 10.4 AI Demand / Replenishment Формулы (текущая реализация)

1. Reorder point engine:

$$
R=V\cdot L\cdot(1+0.15)
$$

где $V$ — дневной burn-rate, $L$ — lead time.

2. Time-to-empty:

$$
TTE=\frac{I_{current}}{V}
$$

3. Urgency:

$$
TTE\le 1.3L \Rightarrow CRITICAL,\quad TTE\le 2.0L \Rightarrow WARNING
$$

4. Look-ahead target:

$$
Target=\max\left(SafetyLevel,\left\lceil FutureDemand\cdot 1.15\right\rceil\right)
$$

5. Factory convoy estimator:

$$
ConvoyCount=\left\lceil\frac{TotalVU}{400}\right\rceil
$$

6. EMA lane dampening:

$$
EMA_{new}=0.8\cdot EMA_{old}+0.2\cdot Transit_{raw}
$$

с propagation при изменении >15% и cooldown не менее 1 часа.

### 10.5 Финансовый Контур: Split + Refund + Settlement (текущая реализация)

1. Split в minor units:

$$
P_{total,minor}=P_{amount}\cdot 100
$$

$$
P_{platform}=\left\lfloor\frac{P_{total,minor}\cdot feeBP}{10000}\right\rfloor
$$

$$
P_{supplier}=P_{total,minor}-P_{platform}
$$

2. `feeBP` берется из platform config (`platform_fee_percent * 100`), дефолт 0 (zero-fee era).

3. Refund reversal применяет ту же basis-point декомпозицию (симметрично split).

4. Financial analytics settlement ratio:

$$
SettlementRate=\frac{TotalRevenue-CashPending}{TotalRevenue}
$$

**Future embodiment (не runtime)**
1. Региональный налоговый коэффициент вида $T_i=S_i\cdot\tau_{region}$ как автоматически применяемая вычислительная модель в платежном контуре пока не закодирован.

### 10.6 Adaptive Route Re-Optimization / Freeze Governance

**Реализовано**
1. Reassignment является детерминированным guard-процессом, а не вероятностной экспоненциальной функцией:

$$
ReassignAllowed \iff \neg FreezeLocked \land StateReassignable \land \neg Sealed \land CapacityOK
$$

2. Freeze lock в AI worker:

$$
t_{exp}=t_{acq}+\max(ttl_{event},300\text{s})
$$

если ttl не задан, применяется дефолт 5 минут.

**Future embodiment (не runtime)**
1. Вероятностный импульс reroute:

$$
P_{reroute}=1-\exp\left(-\frac{\Delta t_{delay}}{T_{buffer}}\right)
$$

может быть добавлен как расширение, но в текущем backend-пути не используется.

### 10.7 Payload Sealing Idempotency (текущая реализация)

1. Клиентский ключ идемпотентности:

$$
K=\text{"payload-"}+action+\text{"-"}+entityId
$$

2. Redis lock key:

$$
L=\text{"idem:"}+K+\text{":lock"},\quad TTL(L)=30\text{s}
$$

3. Replay-cache:

$$
TTL(Cache(K))=24\text{h}
$$

4. UI seal gate:

$$
SealEnabled \iff \forall i\in Manifest(order): scanned_i=1
$$

**Future embodiment (не runtime)**
1. Bloom-filter edge dedupe для сканов в текущем payload-terminal не используется.

### 10.8 Матрица Статуса Формул (IP Hygiene)

1. IMPLEMENTED NOW: H3 distance+load penalty, 0.95 capacity buffer, Kahan volume sum, chunk ceil split, Clarke-Wright savings + 2-opt, telemetry threshold gate (15s/20m/15deg), geofence 100m, reorder/urgency formulas, EMA lane dampening, basis-point split/refund, settlement ratio, freeze-lock TTL fallback, Redis idempotency TTL formulas.
1. FUTURE EMBODIMENT: cellular friction summation $\sum\Omega(h_i)$, acceleration-based dead reckoning, regional tax coefficient $\tau_{region}$ as runtime settlement term, exponential reroute probability $P_{reroute}$, Bloom-filter scan dedupe.
