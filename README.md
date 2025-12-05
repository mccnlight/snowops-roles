# Snowops Roles API

API для управления ролями, организациями, пользователями и водителями.

## Требования

- Go 1.25.0+
- PostgreSQL
- Переменная окружения `DB_DSN` (connection string для PostgreSQL)
- Переменная окружения `JWT_SECRET` (секретный ключ для JWT токенов)

## Установка и запуск

1. Установите зависимости:
```bash
go mod download
```

2. Создайте файл `.env`:
```env
DB_DSN=postgres://user:password@host:port/dbname?sslmode=disable&TimeZone=Asia/Almaty
JWT_SECRET=your-secret-key
APP_PORT=7070
```

3. Запустите приложение:
```bash
go run cmd/Snowops-roles/main.go
```

## Эндпоинты API

Все эндпоинты под `/roles` требуют JWT аутентификацию через заголовок `Authorization: Bearer <token>`.

### Health Check

#### GET /healthz
Проверка работоспособности сервера (без аутентификации)

**Входные данные:** Нет

**Выходные данные:**
```json
{
  "status": "ok"
}
```

---

### Организации (Organizations)

#### GET /roles/organizations
Получить список организаций (доступ зависит от роли)

**Входные данные:** Нет (параметры берутся из JWT токена)

**Выходные данные:**
```json
{
  "organizations": [
    {
      "id": "uuid",
      "name": "Название организации",
      "type": "KGU_ZKH" | "TOO" | "CONTRACTOR" | "AKIMAT",
      "bin": "123456789012",
      "headFullName": "ФИО руководителя",
      "address": "Адрес",
      "phone": "+77001234567",
      "parentOrgID": "uuid" | null,
      "isActive": true,
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Доступ:**
- `AKIMAT_ADMIN`: все активные организации
- `KGU_ZKH_ADMIN`: своя организация + подрядчики
- `TOO_ADMIN`: только своя организация
- `CONTRACTOR_ADMIN`: только своя организация
- `DRIVER`: запрещено

---

#### POST /roles/organizations
Создать новую организацию и администратора для неё

**Входные данные:**
```json
{
  "name": "Название организации",        // обязательно
  "type": "KGU_ZKH" | "TOO" | "CONTRACTOR",  // обязательно
  "bin": "123456789012",
  "headFullName": "ФИО руководителя",
  "address": "Адрес",
  "phone": "+77001234567",
  "adminFullName": "ФИО администратора",
  "adminPhone": "+77001234567",          // обязательно
  "adminPassword": "пароль"                // опционально
}
```

**Выходные данные:**
```json
{
  "organization": {
    "id": "uuid",
    "name": "Название организации",
    "type": "KGU_ZKH" | "TOO" | "CONTRACTOR",
    "bin": "123456789012",
    "headFullName": "ФИО руководителя",
    "address": "Адрес",
    "phone": "+77001234567",
    "parentOrgID": "uuid",
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  },
  "admin": {
    "id": "uuid",
    "phone": "+77001234567",
    "role": "KGU_ZKH_ADMIN" | "TOO_ADMIN" | "CONTRACTOR_ADMIN",
    "organizationID": "uuid",
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Доступ:**
- `AKIMAT_ADMIN`: может создавать KGU_ZKH или TOO
- `KGU_ZKH_ADMIN`: может создавать CONTRACTOR
- Остальные: запрещено

---

#### GET /roles/organizations/:id
Получить информацию об организации по ID

**Входные данные:** ID в URL параметре

**Выходные данные:**
```json
{
  "organization": {
    "id": "uuid",
    "name": "Название организации",
      "type": "KGU_ZKH" | "TOO" | "CONTRACTOR" | "AKIMAT",
    "bin": "123456789012",
    "headFullName": "ФИО руководителя",
    "address": "Адрес",
    "phone": "+77001234567",
    "parentOrgID": "uuid" | null,
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Доступ:**
- `AKIMAT_ADMIN`: любая организация
- `KGU_ZKH_ADMIN`: своя организация или подрядчики
- `CONTRACTOR_ADMIN`: только своя организация
- `DRIVER`: запрещено

---

#### PUT /roles/organizations/:id
Обновить информацию об организации

**Входные данные:** ID в URL параметре, тело запроса (все поля опциональны):
```json
{
  "name": "Новое название",
  "type": "KGU_ZKH" | "CONTRACTOR",
  "bin": "123456789013",
  "head_full_name": "Новое ФИО руководителя",
  "address": "Новый адрес",
  "phone": "+77001234568"
}
```

**Выходные данные:**
```json
{
  "organization": {
    "id": "uuid",
    "name": "Новое название",
    "type": "KGU_ZKH" | "CONTRACTOR",
    "bin": "123456789013",
    "headFullName": "Новое ФИО руководителя",
    "address": "Новый адрес",
    "phone": "+77001234568",
    "parentOrgID": "uuid" | null,
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Доступ:**
- `AKIMAT_ADMIN`: любая организация
- `KGU_ZKH_ADMIN`: своя организация или подрядчики
- `CONTRACTOR_ADMIN`: только своя организация
- `DRIVER`: запрещено

---

#### DELETE /roles/organizations/:id
Удалить организацию (деактивирует организацию, всех пользователей и водителей)

**Входные данные:** ID в URL параметре

**Выходные данные:** HTTP 204 No Content

**Доступ:**
- `AKIMAT_ADMIN`: любая организация
- `KGU_ZKH_ADMIN`: своя организация или подрядчики
- `CONTRACTOR_ADMIN`: только своя организация
- `DRIVER`: запрещено

---

### Пользователи (Users)

#### GET /roles/users?phone=...&login=...
Найти пользователя по телефону или логину

**Входные данные:** Query параметры (`phone` или `login`, хотя бы один обязателен)

**Пример:** `GET /roles/users?phone=+77001234567`

**Выходные данные:**
```json
{
  "user": {
    "id": "uuid",
    "phone": "+77001234567",
    "role": "AKIMAT_ADMIN" | "KGU_ZKH_ADMIN" | "CONTRACTOR_ADMIN" | "DRIVER",
    "login": "login" | null,
    "passwordHash": null,
    "organizationID": "uuid" | null,
    "driverID": "uuid" | null,
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

---

#### GET /roles/users/:id
Получить информацию о пользователе по ID

**Входные данные:** ID в URL параметре

**Выходные данные:**
```json
{
  "user": {
    "id": "uuid",
    "phone": "+77001234567",
    "role": "AKIMAT_ADMIN" | "KGU_ZKH_ADMIN" | "CONTRACTOR_ADMIN" | "DRIVER",
    "login": "login" | null,
    "passwordHash": null,
    "organizationID": "uuid" | null,
    "driverID": "uuid" | null,
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Доступ:**
- Любой пользователь: может получить свои данные
- `AKIMAT_ADMIN`: любой пользователь
- `KGU_ZKH_ADMIN`: пользователи своей организации и подрядчиков
- `CONTRACTOR_ADMIN`: пользователи своей организации

---

#### PUT /roles/users/:id
Обновить информацию о пользователе

**Входные данные:** ID в URL параметре, тело запроса (все поля опциональны):
```json
{
  "phone": "+77001234568",
  "login": "новый_логин",
  "password": "новый_пароль",
  "role": "KGU_ZKH_ADMIN" | "CONTRACTOR_ADMIN" | "DRIVER",
  "organization_id": "uuid",
  "driver_id": "uuid"
}
```

**Выходные данные:**
```json
{
  "user": {
    "id": "uuid",
    "phone": "+77001234568",
    "role": "KGU_ZKH_ADMIN",
    "login": "новый_логин",
    "passwordHash": null,
    "organizationID": "uuid",
    "driverID": "uuid" | null,
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Примечание:** Пароль автоматически хешируется через bcrypt

**Доступ:**
- Любой пользователь: может обновить свои данные
- `AKIMAT_ADMIN`: любой пользователь
- `KGU_ZKH_ADMIN`: пользователи своей организации и подрядчиков
- `CONTRACTOR_ADMIN`: пользователи своей организации

---

### Водители (Drivers)

#### GET /roles/drivers
Получить список водителей (доступ зависит от роли)

**Входные данные:** Нет (параметры берутся из JWT токена)

**Выходные данные:**
```json
{
  "drivers": [
    {
      "id": "uuid",
      "contractorID": "uuid" | null,
      "fullName": "ФИО водителя",
      "iin": "123456789012",
      "birthYear": 1990,
      "phone": "+77001234567",
      "isActive": true,
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Доступ:**
- `AKIMAT_ADMIN`: все активные водители
- `KGU_ZKH_ADMIN`: водители своей организации и подрядчиков
- `CONTRACTOR_ADMIN`: водители своей организации
- `DRIVER`: запрещено

---

#### POST /roles/drivers
Создать нового водителя и пользователя для него

**Входные данные:**
```json
{
  "fullName": "ФИО водителя",    // обязательно
  "iin": "123456789012",          // обязательно
  "birthYear": 1990,               // обязательно
  "phone": "+77001234567"          // обязательно
}
```

**Выходные данные:**
```json
{
  "driver": {
    "id": "uuid",
    "contractorID": "uuid",
    "fullName": "ФИО водителя",
    "iin": "123456789012",
    "birthYear": 1990,
    "phone": "+77001234567",
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  },
  "user": {
    "id": "uuid",
    "phone": "+77001234567",
    "role": "DRIVER",
    "organizationID": "uuid",
    "driverID": "uuid",
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Доступ:** Только `CONTRACTOR_ADMIN`

---

#### GET /roles/drivers/:id
Получить информацию о водителе по ID

**Входные данные:** ID в URL параметре

**Выходные данные:**
```json
{
  "driver": {
    "id": "uuid",
    "contractorID": "uuid" | null,
    "fullName": "ФИО водителя",
    "iin": "123456789012",
    "birthYear": 1990,
    "phone": "+77001234567",
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Доступ:**
- `AKIMAT_ADMIN`: любой водитель
- `KGU_ZKH_ADMIN`: водители подрядчиков
- `CONTRACTOR_ADMIN`: водители своей организации

---

#### PUT /roles/drivers/:id
Обновить информацию о водителе

**Входные данные:** ID в URL параметре, тело запроса (все поля опциональны):
```json
{
  "fullName": "Новое ФИО",
  "phone": "+77001234568",
  "birthYear": 1991,
  "iin": "123456789013"
}
```

**Выходные данные:**
```json
{
  "driver": {
    "id": "uuid",
    "contractorID": "uuid" | null,
    "fullName": "Новое ФИО",
    "iin": "123456789013",
    "birthYear": 1991,
    "phone": "+77001234568",
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Доступ:**
- `AKIMAT_ADMIN`: любой водитель
- `KGU_ZKH_ADMIN`: водители подрядчиков
- `CONTRACTOR_ADMIN`: водители своей организации

---

#### DELETE /roles/drivers/:id
Удалить водителя (деактивирует водителя и связанного пользователя)

**Входные данные:** ID в URL параметре

**Выходные данные:** HTTP 204 No Content

**Доступ:**
- `AKIMAT_ADMIN`: любой водитель
- `KGU_ZKH_ADMIN`: водители подрядчиков
- `CONTRACTOR_ADMIN`: водители своей организации

---

### Транспорт (Vehicles)

#### GET /roles/vehicles?only_active=true
Получить список транспорта.

**Доступ:**
- `AKIMAT_ADMIN`: весь транспорт
- `KGU_ZKH_ADMIN`: транспорт своих подрядчиков
- `CONTRACTOR_ADMIN`: транспорт своей организации

```json
{
  "vehicles": [
    {
      "ID": "0c0c4b4a-5b3e-4c39-b5e5-a3b19a2f1593",
      "ContractorID": "2a1d0a26-7b3d-4132-8f2e-7acd2f4b0da8",
      "PlateNumber": "777ABC01",
      "Brand": "KamAZ",
      "Model": "6520",
      "Color": "Orange",
      "Year": 2022,
      "BodyVolumeM3": 12.5,
      "DriverID": "ae7f5f3a-1a3f-4c90-b8d1-819ffffff0a1",
      "PhotoURL": "https://pub-xxx.r2.dev/snowops-files/vehicles/uuid.png",
      "IsActive": true,
      "CreatedAt": "2025-01-10T07:30:00Z",
      "UpdatedAt": "2025-01-15T08:00:00Z"
    }
  ]
}
```

#### POST /roles/vehicles
Создаёт транспорт (только `CONTRACTOR_ADMIN`). Принимается **только файл** `photo` (multipart/form-data). Поле `photo_url` в запросе не поддерживается — итоговый URL вернётся в ответе после загрузки в R2.

**Входные данные:** multipart/form-data (все поля обязательны, кроме `driver_id` и `is_active`):
- `plate_number` (string, обязательно)
- `brand` (string, обязательно)
- `model` (string, обязательно)
- `color` (string, обязательно)
- `year` (integer, обязательно)
- `body_volume_m3` (float, обязательно)
- `photo` (file, обязательно) — файл изображения (максимум 10MB)
- `driver_id` (string, UUID, опционально)
- `is_active` (boolean, опционально, по умолчанию `true`)

**Пример запроса (curl):**
```bash
curl -X POST http://localhost:7070/roles/vehicles \
  -H "Authorization: Bearer <jwt>" \
  -F "plate_number=888XYZ01" \
  -F "brand=HOWO" \
  -F "model=T5G" \
  -F "color=Blue" \
  -F "year=2021" \
  -F "body_volume_m3=10.2" \
  -F "photo=@/path/to/image.jpg"
```

**Выходные данные:**
```json
{
  "vehicle": {
    "ID": "dd9f1a44-54a0-4c7b-9a40-0c59f4b91d2a",
    "ContractorID": "2a1d0a26-7b3d-4132-8f2e-7acd2f4b0da8",
    "PlateNumber": "888XYZ01",
    "Brand": "HOWO",
    "Model": "T5G",
    "Color": "Blue",
    "Year": 2021,
    "BodyVolumeM3": 10.2,
    "PhotoURL": "https://pub-xxx.r2.dev/snowops-files/vehicles/uuid.jpg",
    "DriverID": null,
    "IsActive": true,
    "CreatedAt": "2025-01-15T10:30:00Z",
    "UpdatedAt": "2025-01-15T10:30:00Z"
  }
}
```

#### GET /roles/vehicles/:id
Подробнее о транспортном средстве. Доступ как в `GET /roles/vehicles`.

#### PATCH /roles/vehicles/:id
Изменить свойства транспорта или привязать водителя.

**Входные данные:** ID в URL параметре, тело запроса multipart/form-data (все поля опциональны):
- `plate_number` (string, опционально)
- `brand` (string, опционально)
- `model` (string, опционально)
- `color` (string, опционально)
- `year` (integer, опционально)
- `body_volume_m3` (float, опционально)
- `photo` (file, опционально) — файл изображения для замены фото (максимум 10MB). Если не указан, фото останется прежним.
- `driver_id` (string, UUID, опционально) — пустая строка для отвязки водителя
- `is_active` (boolean, опционально)

**Важно:** Фото можно обновить только файлом `photo` (multipart/form-data). Поле `photo_url` в запросе не поддерживается.

**Пример запроса (curl) - обновление только цвета:**
```bash
curl -X PATCH http://localhost:7070/roles/vehicles/{vehicle-id} \
  -H "Authorization: Bearer <jwt>" \
  -F "color=White"
```

**Пример запроса (curl) - обновление цвета и фото:**
```bash
curl -X PATCH http://localhost:7070/roles/vehicles/{vehicle-id} \
  -H "Authorization: Bearer <jwt>" \
  -F "color=White" \
  -F "photo=@/path/to/new-image.jpg"
```

**Выходные данные:**
```json
{
  "vehicle": {
    "ID": "0c0c4b4a-5b3e-4c39-b5e5-a3b19a2f1593",
    "ContractorID": "2a1d0a26-7b3d-4132-8f2e-7acd2f4b0da8",
    "PlateNumber": "999ABC01",
    "Brand": "KamAZ",
    "Model": "6520",
    "Color": "White",
    "Year": 2023,
    "BodyVolumeM3": 11.0,
    "DriverID": "ae7f5f3a-1a3f-4c90-b8d1-819ffffff0a1",
    "PhotoURL": "https://pub-xxx.r2.dev/snowops-files/vehicles/uuid.jpg",
    "IsActive": true,
    "CreatedAt": "2025-01-10T07:30:00Z",
    "UpdatedAt": "2025-01-15T08:00:00Z"
  }
}
```

**Доступ:** Только `CONTRACTOR_ADMIN`

#### DELETE /roles/vehicles/:id
Soft-delete: `is_active = false`, привязанный водитель снимается. Доступ только у `CONTRACTOR_ADMIN`.

---

## Роли и типы организаций

### Роли пользователей
- `AKIMAT_ADMIN` - администратор акимата (высший уровень доступа)
- `KGU_ZKH_ADMIN` - администратор KGU ZKH
- `TOO_ADMIN` - техническое ТОО (полигоны и камеры)
- `CONTRACTOR_ADMIN` - администратор подрядчика
- `DRIVER` - водитель

### Типы организаций
- `AKIMAT` - акимат
- `KGU_ZKH` - KGU ZKH (муниципальная организация ЖКХ)
- `TOO` - техническое ТОО
- `CONTRACTOR` - подрядчик

## Аутентификация

Все эндпоинты под `/roles` требуют JWT токен в заголовке:
```
Authorization: Bearer <token>
```

JWT токен должен содержать следующие claims:
- `user_id` - ID пользователя
- `role` - роль пользователя
- `organization_id` - ID организации пользователя

## Ошибки

Все эндпоинты могут возвращать следующие ошибки:

- **400 Bad Request** - неверные входные данные
- **401 Unauthorized** - отсутствует аутентификация или неверный токен
- **403 Forbidden** - недостаточно прав доступа
- **404 Not Found** - ресурс не найден
- **500 Internal Server Error** - внутренняя ошибка сервера

**Формат ошибки:**
```json
{
  "error": "описание ошибки"
}
```

