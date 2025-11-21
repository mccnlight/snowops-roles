# Отсутствующие функции по ТЗ

## Анализ всех сервисов на соответствие требованиям ТЗ

### ✅ РЕАЛИЗОВАНО

#### 1. snowops-roles
- ✅ Управление ролями и организациями
- ✅ Иерархия создания
- ✅ Блокировка пользователей с причиной
- ✅ Автогенерация логинов/паролей

#### 2. snowops-operations-service
- ✅ Управление участками уборки (`cleaning_areas`)
- ✅ Управление полигонами (`polygons`)
- ✅ Управление камерами (`cameras`)
- ✅ Мониторинг техники в реальном времени
- ✅ GPS-симулятор
- ✅ Доступ для `TOO_ADMIN` (LANDFILL_ADMIN) к полигонам и камерам

#### 3. snowops-tickets
- ✅ Управление тикетами
- ✅ Назначения водителей
- ✅ Рейсы (trips) с привязкой к полигонам
- ✅ Обработка событий от камер (LPR/VOLUME)
- ✅ Апелляции водителей

#### 4. snowops-contract-service
- ✅ Контракты KGU ↔ подрядчики (вывоз снега)
- ✅ Отслеживание использования контрактов
- ✅ Привязка тикетов к контрактам

#### 5. snowops-acts-service
- ✅ Акты по вывозу (KGU ↔ подрядчики)
- ✅ Генерация PDF актов
- ✅ Расчёт объёмов и сумм

#### 6. snowops-violations-service
- ✅ Отслеживание нарушений
- ✅ Апелляции по нарушениям

---

## ❌ НЕ РЕАЛИЗОВАНО (критично по ТЗ)

### 1. Контракты KGU ↔ LANDFILL (приём снега)

**Требование ТЗ:**
> Контракты KGU ↔ LANDFILL (приём снега с указанием полигонов и цены за м³)

**Текущее состояние:**
- ❌ В `snowops-contract-service` поддерживаются только контракты с `contractor_id` (подрядчики)
- ❌ Нет поддержки контрактов с `landfill_id` или `polygon_id`
- ❌ Нет типа контракта для приёма снега

**Что нужно добавить:**
1. В `snowops-contract-service`:
   - Добавить поле `contract_type` в таблицу `contracts` (`CONTRACTOR_SERVICE` / `LANDFILL_SERVICE`)
   - Добавить поле `landfill_id` (опционально, для контрактов приёма)
   - Добавить поле `polygon_ids` (массив UUID полигонов) или отдельную таблицу `contract_polygons`
   - Обновить API создания контрактов для поддержки LANDFILL
   - Обновить права доступа (LANDFILL_ADMIN может видеть свои контракты)

**Пример структуры:**
```sql
ALTER TABLE contracts ADD COLUMN contract_type VARCHAR(50) NOT NULL DEFAULT 'CONTRACTOR_SERVICE';
ALTER TABLE contracts ADD COLUMN landfill_id UUID REFERENCES organizations(id);
CREATE TABLE contract_polygons (
  contract_id UUID REFERENCES contracts(id),
  polygon_id UUID REFERENCES polygons(id),
  PRIMARY KEY (contract_id, polygon_id)
);
```

---

### 2. Акты по приёму снега (KGU ↔ LANDFILL)

**Требование ТЗ:**
> Акты по приёму: формировать акты KGU ↔ LANDFILL:
> - считать общий объём принятого снега по полигонам и периодам
> - строить свод по полигонам и подрядчикам
> - отправлять акт оператору полигона на подтверждение

**Текущее состояние:**
- ❌ В `snowops-acts-service` поддерживаются только акты по контрактам с подрядчиками
- ❌ Нет поддержки актов по контрактам приёма (LANDFILL)
- ❌ Нет механизма подтверждения актов LANDFILL

**Что нужно добавить:**
1. В `snowops-acts-service`:
   - Расширить модель `Act` для поддержки актов приёма:
     - Добавить поле `landfill_id` (опционально)
     - Добавить поле `contract_type` или использовать `contract.contract_type`
   - Добавить статусы акта: `GENERATED`, `PENDING_APPROVAL`, `APPROVED`, `REJECTED`
   - Добавить поле `rejection_reason` (текст причины отклонения)
   - Добавить поле `approved_by_org_id` и `approved_by_user_id`
   - Добавить поле `approved_at`
   - Создать эндпоинты:
     - `GET /acts/landfill` - список актов для LANDFILL (только `LANDFILL_ADMIN`)
     - `GET /acts/landfill/:id` - просмотр акта
     - `PUT /acts/landfill/:id/approve` - подтвердить акт
     - `PUT /acts/landfill/:id/reject` - отклонить акт с причиной
   - Обновить логику генерации актов:
     - Для контрактов приёма считать объём по рейсам, где `trip.polygon_id` входит в список полигонов контракта
     - Группировать по полигонам и подрядчикам

**Пример структуры:**
```sql
ALTER TABLE act ADD COLUMN landfill_id UUID REFERENCES organizations(id);
ALTER TABLE act ADD COLUMN contract_type VARCHAR(50) NOT NULL DEFAULT 'CONTRACTOR_SERVICE';
ALTER TABLE act ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'GENERATED';
ALTER TABLE act ADD COLUMN rejection_reason TEXT;
ALTER TABLE act ADD COLUMN approved_by_org_id UUID REFERENCES organizations(id);
ALTER TABLE act ADD COLUMN approved_by_user_id UUID REFERENCES users(id);
ALTER TABLE act ADD COLUMN approved_at TIMESTAMPTZ;
```

**Пример API:**
```json
PUT /acts/landfill/:id/approve
{
  "comment": "Акт подтверждён"
}

PUT /acts/landfill/:id/reject
{
  "reason": "Несоответствие объёмов по полигону №2"
}
```

---

### 3. Журнал приёма снега для LANDFILL

**Требование ТЗ:**
> Журнал приёма снега:
> - просмотр всех заездов на полигоны:
>   - дата/время
>   - полигон
>   - госномер
>   - подрядчик (если привязан)
>   - объём м³
>   - статус распознавания

**Текущее состояние:**
- ✅ Рейсы (`trips`) хранят информацию о заездах на полигоны
- ❌ Нет отдельного эндпоинта для LANDFILL для просмотра журнала приёма
- ❌ Нет фильтрации по полигонам, принадлежащим LANDFILL организации

**Что нужно добавить:**
1. В `snowops-tickets` или создать отдельный эндпоинт:
   - `GET /landfill/trips` или `GET /landfill/reception-journal`
   - Фильтры: `polygon_id`, `date_from`, `date_to`, `contractor_id`, `status`
   - Доступ: только `LANDFILL_ADMIN` и `LANDFILL_USER`
   - Возвращает рейсы, где `trip.polygon_id` принадлежит полигонам LANDFILL организации

**Пример ответа:**
```json
{
  "data": {
    "trips": [
      {
        "id": "...",
        "entry_at": "2025-01-15T10:30:00Z",
        "exit_at": "2025-01-15T10:45:00Z",
        "polygon_id": "...",
        "polygon_name": "Полигон №1",
        "vehicle_plate_number": "KZ 123 ABC",
        "detected_plate_number": "KZ 123 ABC",
        "contractor_id": "...",
        "contractor_name": "TOO Snow Demo",
        "detected_volume_entry": 42.5,
        "detected_volume_exit": 2.1,
        "net_volume_m3": 40.4,
        "status": "OK"
      }
    ],
    "total_volume_m3": 1250.8,
    "total_trips": 31
  }
}
```

---

### 4. Подтверждение актов LANDFILL

**Требование ТЗ:**
> LANDFILL имеет право подтвердить или отклонить акт, добавив комментарий;
> только после подтверждения акт считается окончательно согласованным.

**Текущее состояние:**
- ❌ В `snowops-acts-service` статус акта только `GENERATED`, нет workflow подтверждения
- ❌ Нет эндпоинтов для подтверждения/отклонения актов

**Что нужно добавить:**
1. В `snowops-acts-service`:
   - Добавить статусы: `GENERATED`, `PENDING_APPROVAL`, `APPROVED`, `REJECTED`
   - При создании акта по контракту приёма устанавливать статус `PENDING_APPROVAL`
   - Добавить эндпоинты (см. раздел 2)
   - Обновить PDF генератор для отображения статуса акта

---

### 5. Обновление ролей в других сервисах

**Проблема:**
- В `snowops-contract-service` используется `TOO_ADMIN` вместо `LANDFILL_ADMIN`
- В `snowops-operations-service` используется `TOO_ADMIN` вместо `LANDFILL_ADMIN`
- В `snowops-acts-service` нет поддержки `LANDFILL_ADMIN`

**Что нужно исправить:**
1. Обновить все сервисы для поддержки `LANDFILL_ADMIN` и `LANDFILL_USER`:
   - `snowops-contract-service`: добавить `LANDFILL_ADMIN` в проверки прав
   - `snowops-operations-service`: заменить `TOO_ADMIN` на `LANDFILL_ADMIN` или поддерживать оба
   - `snowops-acts-service`: добавить поддержку `LANDFILL_ADMIN` для подтверждения актов
   - `snowops-tickets`: добавить поддержку `LANDFILL_ADMIN` для просмотра журнала приёма

---

### 6. Дашборды и аналитика

**Требование ТЗ:**
> Дашборды с общей статистикой, мониторинг и аналитика для всех ролей

**Текущее состояние:**
- ✅ Мониторинг техники в реальном времени (`snowops-operations-service`)
- ❌ Нет дашбордов с общей статистикой
- ❌ Нет аналитики по объёмам вывоз/приём
- ❌ Нет отчётов по периодам

**Что нужно добавить:**
1. Создать новый сервис `snowops-analytics-service` или добавить в существующие:
   - Дашборд для Акимата (общая статистика по городу)
   - Дашборд для КГУ (статистика по подрядчикам и LANDFILL)
   - Дашборд для подрядчика (статистика по своей организации)
   - Дашборд для LANDFILL (статистика по приёму снега)
   - Отчёты по объёмам вывоз/приём
   - Графики и визуализации

---

## 📋 ПРИОРИТЕТНЫЙ СПИСОК РЕАЛИЗАЦИИ

### Критично (для работы системы):

1. **Контракты KGU ↔ LANDFILL** (`snowops-contract-service`)
   - Добавить тип контракта и поддержку LANDFILL
   - Оценка: 4-6 часов

2. **Акты по приёму снега** (`snowops-acts-service`)
   - Расширить модель актов для LANDFILL
   - Оценка: 6-8 часов

3. **Подтверждение актов LANDFILL** (`snowops-acts-service`)
   - Добавить workflow подтверждения/отклонения
   - Оценка: 4-6 часов

4. **Журнал приёма снега** (`snowops-tickets` или новый эндпоинт)
   - Эндпоинт для LANDFILL для просмотра заездов
   - Оценка: 2-4 часа

5. **Обновление ролей** (все сервисы)
   - Заменить `TOO_ADMIN` на `LANDFILL_ADMIN` или поддерживать оба
   - Оценка: 2-3 часа

### Важно (для полноты функциональности):

6. **Дашборды и аналитика**
   - Новый сервис или расширение существующих
   - Оценка: 16-24 часа

---

## 📝 ДЕТАЛЬНЫЕ ТРЕБОВАНИЯ К РЕАЛИЗАЦИИ

### 1. Контракты KGU ↔ LANDFILL

**Файлы для изменения:**
- `snowops-contract-service/internal/db/migrations.go` - добавить поля
- `snowops-contract-service/internal/model/domain.go` - обновить модель
- `snowops-contract-service/internal/service/contract_service.go` - обновить логику
- `snowops-contract-service/internal/http/handler.go` - обновить API

**Новые поля в Contract:**
```go
type Contract struct {
    // ... существующие поля
    ContractType string    // "CONTRACTOR_SERVICE" | "LANDFILL_SERVICE"
    LandfillID   *uuid.UUID // для контрактов приёма
}
```

**Новая таблица:**
```sql
CREATE TABLE contract_polygons (
    contract_id UUID REFERENCES contracts(id) ON DELETE CASCADE,
    polygon_id UUID REFERENCES polygons(id) ON DELETE CASCADE,
    PRIMARY KEY (contract_id, polygon_id)
);
```

**API изменения:**
```json
POST /contracts
{
  "contract_type": "LANDFILL_SERVICE",
  "landfill_id": "uuid",
  "polygon_ids": ["uuid1", "uuid2"],
  "price_per_m3": 500.00,
  ...
}
```

---

### 2. Акты по приёму снега

**Файлы для изменения:**
- `snowops-acts-service/internal/db/migrations.go` - добавить поля
- `snowops-acts-service/internal/model/act.go` - обновить модель
- `snowops-acts-service/internal/service/act_service.go` - обновить логику генерации
- `snowops-acts-service/internal/http/handler.go` - добавить эндпоинты

**Новые статусы:**
```go
const (
    ActStatusGenerated      = "GENERATED"
    ActStatusPendingApproval = "PENDING_APPROVAL"
    ActStatusApproved       = "APPROVED"
    ActStatusRejected       = "REJECTED"
)
```

**Новые эндпоинты:**
- `GET /acts/landfill` - список актов для LANDFILL
- `GET /acts/landfill/:id` - просмотр акта
- `PUT /acts/landfill/:id/approve` - подтвердить
- `PUT /acts/landfill/:id/reject` - отклонить

**Логика генерации:**
- Для контрактов `LANDFILL_SERVICE`:
  - Найти все рейсы, где `trip.polygon_id` входит в `contract_polygons`
  - Сгруппировать по полигонам и подрядчикам
  - Рассчитать объём и стоимость
  - Создать акт со статусом `PENDING_APPROVAL`

---

### 3. Журнал приёма снега

**Файлы для изменения:**
- `snowops-tickets/internal/http/handler.go` - добавить эндпоинт
- `snowops-tickets/internal/service/trip_service.go` - добавить метод

**Новый эндпоинт:**
```
GET /landfill/reception-journal
Query params:
  - polygon_id (optional)
  - date_from (optional)
  - date_to (optional)
  - contractor_id (optional)
  - status (optional)
```

**Логика:**
- Получить `organization_id` из JWT (должен быть LANDFILL)
- Найти все полигоны, принадлежащие этой организации
- Найти все рейсы, где `trip.polygon_id` входит в список полигонов
- Применить фильтры
- Вернуть список с группировкой по полигонам

---

## 🎯 ИТОГОВАЯ ОЦЕНКА

**Общая готовность системы: ~75%**

**Реализовано:**
- ✅ Управление ролями и организациями
- ✅ Управление участками, полигонами, камерами
- ✅ Управление тикетами и рейсами
- ✅ Контракты KGU ↔ подрядчики
- ✅ Акты по вывозу (KGU ↔ подрядчики)
- ✅ Мониторинг техники

**Не реализовано (критично):**
- ❌ Контракты KGU ↔ LANDFILL
- ❌ Акты по приёму (KGU ↔ LANDFILL)
- ❌ Подтверждение актов LANDFILL
- ❌ Журнал приёма снега для LANDFILL
- ❌ Обновление ролей в других сервисах

**Оценка времени на реализацию: 18-27 часов**

