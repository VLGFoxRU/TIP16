# ЭФМО-01-25 Буров М.А. ПР16

# Описание проекта
Интеграционное тестирование API. Использование Docker для тестовой БД

# Требования к проекту
* Go 1.25+
* Git

# Версия Go
<img width="317" height="55" alt="image" src="https://github.com/user-attachments/assets/43f9087b-95b9-4c7d-86e9-746258c45c63" />

# Цели:
-	Освоить интеграционное тестирование REST API: проверка «маршрут → хендлер → сервис → репозиторий → реальная БД».
-	Научиться поднимать изолированную тестовую среду БД в Docker.
-	Освоить 2 подхода к инфраструктуре тестов:
	  A. Локальная среда через docker-compose (просто и наглядно).
	  B. Программный подъём контейнеров через testcontainers-go (изолировано и удобно для CI).
-	Научиться инициализировать схему БД (миграции/auto-migrate), сидировать тестовые данные, очищать окружение.
-	Внедрить интеграционные проверки CRUD-эндпоинтов (статусы, заголовки, JSON-ответы, эффекты в БД).


# Структура проекта
Дерево структуры проекта: 
```
pz16-integration/
├── cmd/api/
│   └── main.go 
│
├── internal/
│   ├── models/
│   │   └── note.go
│   │
│   ├── repo/
│   │   └── postgres.go
│   │
│   ├── service/
│   │   └── service.go
│   │
│   ├── httpapi/
│   │   └── handlers.go 
│   │
│   └── db/
│       └── migrate.go 
│
├── integration/
│   └── notes_integration_test.go 
│
├── docker-compose.yml
├── README.md 
├── go.mod
└── go.sum

```

# Скриншоты

<img width="1399" height="182" alt="image" src="https://github.com/user-attachments/assets/93d8b95b-d3ee-49ae-b501-d95b4aaf1ffa" />

<img width="1251" height="116" alt="image" src="https://github.com/user-attachments/assets/baf9250a-c541-4b8b-9157-c73ffe5ae41a" />

<img width="1186" height="443" alt="image" src="https://github.com/user-attachments/assets/5ccddf7b-75bc-4aae-881a-a4f0757b043c" />

<img width="1274" height="959" alt="image" src="https://github.com/user-attachments/assets/df2e1a89-e054-4ffd-9ff6-4a34858b83e8" />

# Краткие выводы

В рамках практической работы были выполнены интеграционные проверки REST API заметок с реальной PostgreSQL, поднятой в Docker.
Схема БД инициализируется автоматически через миграции.
Для изоляции тестов применяется очистка таблицы notes перед каждым тестом, поэтому результаты не зависят от порядка выполнения.
Отдельно проверены граничные и ошибочные случаи: 404 для отсутствующих сущностей и 400 для некорректных запросов.
  
# Ответы на контрольные вопросы

1.	Чем интеграционные тесты отличаются от модульных? Когда целесообразно использовать каждый вид?

Unit-тесты:

- Тестируют отдельные функции в изоляции
- Используют mock'и и stub'ы вместо реальных зависимостей
- Быстрые, независят от внешних сервисов
- Проверяют внутреннюю логику

Интеграционные тесты:

- Тестируют взаимодействие нескольких компонентов вместе
- Используют реальные зависимости (БД, HTTP, файловая система)
- Медленнее, но более репрезентативны
- Проверяют, что система работает как целое

2.	Зачем использовать Docker для БД, если локально уже есть PostgreSQL/MySQL?

- Повторяемость: каждый тест/разработчик/CI получает одинаковую среду.
- Изоляция: не ломаем локальную БД и не завязаны на внешний сервис.
- Чистота: можно гарантировано стартовать с «чистой» схемой/данными.

3.	Плюсы и минусы подходов docker-compose и testcontainers-go для интеграций.

- docker-compose
Вы руками (или через make) поднимаете контейнер БД. Тесты подключаются к известному DSN. Простой старт, минимум магии, чуть сложнее в CI.

- testcontainers-go
Тесты сами поднимают и останавливают контейнер БД через код Go. Отлично подходит для CI, изолированно и «самодостаточно», но требует зависимости и немного большего кода.

4.	Как обеспечить независимость интеграционных тестов (данные, транзакции, порядок)?

### Подход 1: TRUNCATE перед каждым тестом (ПЗ №16)

```
func setupTestServer(t *testing.T) *TestHelper {
    // Очищаем все таблицы
    dbx.Exec("TRUNCATE TABLE notes RESTART IDENTITY CASCADE")
    
    t.Cleanup(func() {
        // Снова чистим (если тест упал)
        dbx.Exec("TRUNCATE TABLE notes RESTART IDENTITY CASCADE")
    })
    // ...
}
```

**Плюсы:** Просто, надежно, гарантирует чистое состояние
**Минусы:** Медленно для больших таблиц, нельзя использовать данные между тестами

### Подход 2: Транзакции (откат всех изменений)

```
func setupTestServer(t *testing.T) *TestHelper {
    // Начинаем транзакцию
    tx, _ := dbx.Begin()
    
    t.Cleanup(func() {
        // Откатываем все изменения теста
        tx.Rollback()
    })
    // Используем tx вместо dbx
    // ...
}
```

**Плюсы:** Очень быстро (нет удаления), изолирует все изменения
**Минусы:** Не работает с `AUTOCOMMIT` операциями, требует переписывания кода

### Подход 3: Отдельные данные для каждого теста

```
func TestCreateNote(t *testing.T) {
    input1 := models.CreateNoteInput{Title: "Note 1", Content: "Content 1"}
    resp1, _ := helper.makeRequest(t, "POST", "/api/notes", input1)
    // Используем только свои данные
    
    input2 := models.CreateNoteInput{Title: "Note 2", Content: "Content 2"}
    resp2, _ := helper.makeRequest(t, "POST", "/api/notes", input2)
    // Проверяем только Note 1 и Note 2, не полагаемся на старые данные
}
```

**Плюсы:** Гарантирует независимость
**Минусы:** Требует дисциплины от разработчика

5.	Как вы бы тестировали пагинацию и фильтрацию в списковых эндпоинтах?

### Пагинация: offset/limit

```
func TestGetNotesPagination(t *testing.T) {
    helper := setupTestServer(t)
    
    // Создаем 15 заметок
    for i := 1; i <= 15; i++ {
        input := models.CreateNoteInput{
            Title:   fmt.Sprintf("Note %d", i),
            Content: fmt.Sprintf("Content %d", i),
        }
        helper.makeRequest(t, "POST", "/api/notes", input)
    }
    
    // Тест 1: первая страница (offset=0, limit=5)
    resp, body := helper.makeRequest(t, "GET", "/api/notes?offset=0&limit=5", nil)
    var page1 struct {
        Notes  []models.Note `json:"notes"`
        Count  int           `json:"count"`
        Offset int           `json:"offset"`
        Limit  int           `json:"limit"`
    }
    json.Unmarshal([]byte(body), &page1)
    
    if page1.Count != 5 {
        t.Errorf("Expected 5 notes, got %d", page1.Count)
    }
    if page1.Notes[0].Title != "Note 15" {
        t.Error("Expected notes in DESC order (newest first)")
    }
    
    // Тест 2: вторая страница (offset=5, limit=5)
    resp2, body2 := helper.makeRequest(t, "GET", "/api/notes?offset=5&limit=5", nil)
    var page2 struct {
        Notes []models.Note `json:"notes"`
        Count int           `json:"count"`
    }
    json.Unmarshal([]byte(body2), &page2)
    
    if page2.Count != 5 {
        t.Errorf("Expected 5 notes on page 2, got %d", page2.Count)
    }
    // Проверяем, что Notes из page2 != Notes из page1
    if page2.Notes[0].ID == page1.Notes[0].ID {
        t.Error("Page 2 should have different notes than page 1")
    }
    
    // Тест 3: за границей (offset=15, limit=5)
    resp3, body3 := helper.makeRequest(t, "GET", "/api/notes?offset=15&limit=5", nil)
    var page3 struct {
        Notes []models.Note `json:"notes"`
        Count int           `json:"count"`
    }
    json.Unmarshal([]byte(body3), &page3)
    
    if page3.Count != 0 {
        t.Errorf("Expected 0 notes beyond limit, got %d", page3.Count)
    }
    
    // Тест 4: limit слишком большой (должна быть капсула)
    resp4, body4 := helper.makeRequest(t, "GET", "/api/notes?offset=0&limit=1000", nil)
    var page4 struct {
        Limit int `json:"limit"`
    }
    json.Unmarshal([]byte(body4), &page4)
    
    if page4.Limit > 100 {
        t.Errorf("Limit should be capped at 100, got %d", page4.Limit)
    }
}
```

### Фильтрация: поиск по title

```
func TestSearchNotes(t *testing.T) {
    helper := setupTestServer(t)
    
    // Создаем заметки с разными заголовками
    notes := []string{
        "golang tutorial",
        "go programming",
        "rust guide",
        "python tips",
    }
    
    for _, title := range notes {
        input := models.CreateNoteInput{Title: title, Content: "Content"}
        helper.makeRequest(t, "POST", "/api/notes", input)
    }
    
    // Тест 1: поиск "go"
    resp, body := helper.makeRequest(t, "GET", "/api/notes?search=go", nil)
    var result struct {
        Notes []models.Note `json:"notes"`
        Count int           `json:"count"`
    }
    json.Unmarshal([]byte(body), &result)
    
    if result.Count != 2 {
        t.Errorf("Expected 2 notes with 'go', got %d", result.Count)
    }
    for _, note := range result.Notes {
        if !strings.Contains(strings.ToLower(note.Title), "go") {
            t.Errorf("Note %q doesn't contain 'go'", note.Title)
        }
    }
    
    // Тест 2: поиск "GO" (case-insensitive)
    resp2, body2 := helper.makeRequest(t, "GET", "/api/notes?search=GO", nil)
    var result2 struct {
        Count int `json:"count"`
    }
    json.Unmarshal([]byte(body2), &result2)
    
    if result2.Count != 2 {
        t.Error("Search should be case-insensitive")
    }
    
    // Тест 3: поиск "python"
    resp3, body3 := helper.makeRequest(t, "GET", "/api/notes?search=python", nil)
    var result3 struct {
        Count int `json:"count"`
    }
    json.Unmarshal([]byte(body3), &result3)
    
    if result3.Count != 1 {
        t.Errorf("Expected 1 note with 'python', got %d", result3.Count)
    }
    
    // Тест 4: поиск несуществующего
    resp4, body4 := helper.makeRequest(t, "GET", "/api/notes?search=nonexistent", nil)
    var result4 struct {
        Count int `json:"count"`
    }
    json.Unmarshal([]byte(body4), &result4)
    
    if result4.Count != 0 {
        t.Error("Search for nonexistent should return 0")
    }
}
```

6.	Какие риски несут «ломающие» миграции и как их безопасно применять в тестах?

## Что такое «ломающая» миграция

```
-- ЛОМАЮЩАЯ: удаляем колонку, теряются данные
ALTER TABLE notes DROP COLUMN content;

-- ЛОМАЮЩАЯ: меняем тип колонки (int → string)
ALTER TABLE notes ALTER COLUMN title TYPE INT;

-- ЛОМАЮЩАЯ: добавляем NOT NULL без default
ALTER TABLE notes ADD COLUMN status VARCHAR NOT NULL;

-- БЕЗОПАСНАЯ: добавляем новую колонку с default
ALTER TABLE notes ADD COLUMN status VARCHAR DEFAULT 'active';

-- БЕЗОПАСНАЯ: переименовываем (разные системы, но данные сохранены)
ALTER TABLE notes RENAME COLUMN content TO description;
```

## Риски ломающих миграций

| Риск | Пример | Последствие |
|------|--------|------------|
| **Потеря данных** | DROP COLUMN | Невозможно восстановить |
| **Несовместимость** | Изменение типа | Приложение падает |
| **Downtime** | LOCK на таблице | Сервис недоступен минуты |
| **Rollback сложный** | Удаление данных | Нельзя откатить |

