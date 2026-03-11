# CrackHash - Распределенная система для взлома хэшей

CrackHash — это распределенная система для взлома MD5-хэшей методом перебора (brute-force) словаря, генерируемого на основе заданного алфавита. Система построена на микросервисной архитектуре с использованием языка Go и состоит из Менеджера и Воркеров.

## Архитектура системы

Система состоит из двух основных компонентов:

1.  **Менеджер (Manager)**:
    *   Принимает REST API запросы от клиентов на взлом хэша.
    *   Сохраняет состояние запроса в оперативной памяти (In-Memory Storage).
    *   Разбивает задачу на подзадачи и асинхронно распределяет их между доступными Воркерами через Internal API.
    *   Принимает результаты от Воркеров, агрегирует их и обновляет статус задачи.
    *   Предоставляет клиенту API для проверки статуса и получения результатов.

2.  **Воркер (Worker)**:
    *   Принимает задачи от Менеджера.
    *   Выполняет итеративную генерацию слов из заданного алфавита. Для экономии памяти и справедливого распределения нагрузки, каждый воркер генерирует слова по принципу "одометра", но вычисляет MD5-хэш только для тех слов, чей порядковый номер соответствует его идентификатору (PartNumber) по модулю общего количества воркеров (TotalParts).
    *   При совпадении хэша сохраняет найденное слово.
    *   По завершении перебора своей части пространства, отправляет результаты обратно Менеджеру.

### Схема взаимодействия (Sequence Diagram)

```mermaid
sequenceDiagram
    participant Client
    participant Manager
    participant Worker 1
    participant Worker 2
    participant Worker 3

    Client->>Manager: POST /api/hash/crack {hash, maxLength}
    Manager-->>Client: 200 OK {requestId}
    
    note over Manager: Создание задачи (Status: IN_PROGRESS)
    
    par Разделение задач
        Manager->>Worker 1: POST /internal/api/worker/hash/crack/task (Part 0/3)
        Manager->>Worker 2: POST /internal/api/worker/hash/crack/task (Part 1/3)
        Manager->>Worker 3: POST /internal/api/worker/hash/crack/task (Part 2/3)
    end
    
    Worker 1-->>Manager: 200 OK (Task accepted)
    Worker 2-->>Manager: 200 OK (Task accepted)
    Worker 3-->>Manager: 200 OK (Task accepted)
    
    note over Worker 1, Worker 3: Асинхронный перебор пространства
    
    Client->>Manager: GET /api/hash/status?requestId=...
    Manager-->>Client: 200 OK {status: "IN_PROGRESS", progress: 0}
    
    Worker 1->>Manager: PATCH /internal/api/manager/hash/crack/request {foundWords}
    note over Manager: Агрегация: Progress 33%
    
    Worker 2->>Manager: PATCH /internal/api/manager/hash/crack/request {foundWords}
    note over Manager: Агрегация: Progress 66%
    
    Worker 3->>Manager: PATCH /internal/api/manager/hash/crack/request {foundWords}
    note over Manager: Агрегация: Progress 100%. Status: READY
    
    Client->>Manager: GET /api/hash/status?requestId=...
    Manager-->>Client: 200 OK {status: "READY", data: ["word"]}
```

## Инструкция по запуску

Для запуска системы необходим установленный Docker и docker-compose.

1.  Склонируйте репозиторий и перейдите в его корневую директорию.
2.  Выполните команду для сборки и запуска контейнеров:
    ```bash
    docker-compose up --build -d
    ```
3.  Система будет запущена:
    *   Менеджер доступен на порту `8080`.
    *   Воркеры доступны на портах `8081`, `8082`, `8083`.

Для остановки системы выполните:
```bash
docker-compose down
```

## Конфигурационные параметры

Настройки системы задаются через переменные окружения в файле `docker-compose.yml`:

### Для Менеджера (manager):
*   `MANAGER_PORT` — порт, на котором запускается менеджер (по умолчанию `8080`).
*   `WORKER_URLS` — список URL воркеров, разделенный запятыми. Менеджер использует этот список для рассылки задач. (Пример: `http://worker1:8081,http://worker2:8081,http://worker3:8081`).
*   `ALPHABET` — алфавит, используемый для генерации слов. (По умолчанию `abcdefghijklmnopqrstuvwxyz0123456789`).

### Для Воркера (worker):
*   `WORKER_PORT` — порт внутреннего сервера воркера (по умолчанию `8081`).
*   `MANAGER_URL` — URL менеджера, на который воркер будет отправлять результаты работы. (Пример: `http://manager:8080`).

## Описание API и примеры запросов

### Public API (Client -> Manager)

#### 1. Запрос на взлом хэша
**POST** `http://localhost:8080/api/hash/crack`

**Request Body:**
```json
{
    "hash": "e2fc714c4727ee9395f324cd2e7f331f", 
    "maxLength": 4
}
```

**Response (200 OK):**
```json
{
    "requestId": "730a04e6-4de9-41f9-9d5b-53b88b17afac"
}
```

#### 2. Проверка статуса задачи
**GET** `http://localhost:8080/api/hash/status?requestId={requestId}`

**Response (В процессе - 200 OK):**
```json
{
    "status": "IN_PROGRESS",
    "progress": 33,
    "data": null
}
```

**Response (Готово - 200 OK):**
```json
{
   "status": "READY",
   "progress": 100,
   "data": ["abcd"]
}
```

**Возможные статусы:**
*   `IN_PROGRESS` — задача выполняется.
*   `READY` — задача успешно завершена.
*   `ERROR` — произошла ошибка при выполнении.

### Internal API (Документация Swagger)

Воркер предоставляет автоматически сгенерированную документацию OpenAPI (Swagger) для своего внутреннего API.
После запуска системы, вы можете просмотреть Swagger UI по адресу:
`http://localhost:8081/swagger/index.html` (для первого воркера).