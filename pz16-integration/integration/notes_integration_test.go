package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"example.com/pz16-integration/internal/db"
	"example.com/pz16-integration/internal/httpapi"
	"example.com/pz16-integration/internal/models"
	"example.com/pz16-integration/internal/repo"
	"example.com/pz16-integration/internal/service"
)

// TestHelper помощник для инициализации тестового сервера
type TestHelper struct {
	db     *sql.DB
	server *httptest.Server
	svc    *service.Service
}

// setupTestServer создает и возвращает тестовый HTTP-сервер с реальной БД
func setupTestServer(t *testing.T) *TestHelper {
	t.Helper()

	// Получаем DSN из переменной окружения
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("DB_DSN environment variable not set. Run: make up && make test")
	}

	// Открываем подключение к БД
	dbx, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Проверяем подключение
	if err := dbx.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Применяем миграции
	db.MustApplyMigrations(dbx)

	// Очищаем БД перед тестом
	if _, err := dbx.Exec("TRUNCATE TABLE notes RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("Failed to truncate table: %v", err)
	}

	// Создаем слои приложения
	noteRepo := repo.NoteRepo{DB: dbx}
	svc := service.Service{Notes: noteRepo}

	// Настраиваем Gin в режиме тестирования
	gin.SetMode(gin.TestMode)

	// Создаем HTTP-сервер
	engine := gin.Default()
	router := httpapi.Router{Service: &svc}
	router.Register(engine)

	server := httptest.NewServer(engine)

	// Регистрируем cleanup для закрытия ресурсов после теста
	t.Cleanup(func() {
		server.Close()
		if _, err := dbx.Exec("TRUNCATE TABLE notes RESTART IDENTITY CASCADE"); err != nil {
			t.Logf("Warning: failed to truncate table: %v", err)
		}
		dbx.Close()
	})

	return &TestHelper{
		db:     dbx,
		server: server,
		svc:    &svc,
	}
}

// makeRequest делает HTTP-запрос и возвращает ответ
func (h *TestHelper) makeRequest(
	t *testing.T,
	method string,
	path string,
	body interface{},
) (*http.Response, string) {
	t.Helper()

	url := h.server.URL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	_ = resp.Body.Close()

	return resp, string(respBody)
}

// ============== ТЕСТЫ ==============

// TestCreateNote тестирует создание новой заметки
func TestCreateNote(t *testing.T) {
	helper := setupTestServer(t)

	input := models.CreateNoteInput{
		Title:   "My First Note",
		Content: "This is a test note",
	}

	resp, body := helper.makeRequest(t, "POST", "/api/notes", input)

	// Проверяем статус код
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
		t.Logf("Response body: %s", body)
	}

	// Парсим ответ
	var note models.Note
	if err := json.Unmarshal([]byte(body), &note); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Проверяем поля ответа
	if note.ID == 0 {
		t.Error("Expected ID to be set")
	}

	if note.Title != input.Title {
		t.Errorf("Expected title %q, got %q", input.Title, note.Title)
	}

	if note.Content != input.Content {
		t.Errorf("Expected content %q, got %q", input.Content, note.Content)
	}

	if note.CreatedAt == "" {
		t.Error("Expected CreatedAt to be set")
	}

	if note.UpdatedAt == "" {
		t.Error("Expected UpdatedAt to be set")
	}
}

// TestCreateNoteWithEmptyTitle тестирует ошибку создания с пустым заголовком
func TestCreateNoteWithEmptyTitle(t *testing.T) {
	helper := setupTestServer(t)

	input := models.CreateNoteInput{
		Title:   "", // Пусто!
		Content: "This should fail",
	}

	resp, body := helper.makeRequest(t, "POST", "/api/notes", input)

	// Ожидаем ошибку валидации
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
		t.Logf("Response body: %s", body)
	}

	// Проверяем, что ошибка содержит "title"
	if !bytes.Contains([]byte(body), []byte("title")) {
		t.Errorf("Expected error about title, got: %s", body)
	}
}

// TestGetNote тестирует получение заметки по ID
func TestGetNote(t *testing.T) {
	helper := setupTestServer(t)

	// Сначала создаем заметку
	createInput := models.CreateNoteInput{
		Title:   "Test Note",
		Content: "Test Content",
	}

	resp, body := helper.makeRequest(t, "POST", "/api/notes", createInput)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create note: %d", resp.StatusCode)
	}

	var created models.Note
	json.Unmarshal([]byte(body), &created)

	// Теперь получаем её
	resp2, body2 := helper.makeRequest(t, "GET", fmt.Sprintf("/api/notes/%d", created.ID), nil)

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp2.StatusCode)
		t.Logf("Response body: %s", body2)
	}

	var fetched models.Note
	json.Unmarshal([]byte(body2), &fetched)

	if fetched.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, fetched.ID)
	}

	if fetched.Title != created.Title {
		t.Errorf("Expected title %q, got %q", created.Title, fetched.Title)
	}
}

// TestGetNoteNotFound тестирует получение несуществующей заметки
func TestGetNoteNotFound(t *testing.T) {
	helper := setupTestServer(t)

	resp, body := helper.makeRequest(t, "GET", "/api/notes/99999", nil)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
		t.Logf("Response body: %s", body)
	}
}

// TestGetNotes тестирует получение списка заметок
func TestGetNotes(t *testing.T) {
	helper := setupTestServer(t)

	// Создаем несколько заметок
	for i := 1; i <= 3; i++ {
		input := models.CreateNoteInput{
			Title:   fmt.Sprintf("Note %d", i),
			Content: fmt.Sprintf("Content %d", i),
		}
		resp, _ := helper.makeRequest(t, "POST", "/api/notes", input)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to create note %d", i)
		}
	}

	// Получаем список
	resp, body := helper.makeRequest(t, "GET", "/api/notes?offset=0&limit=10", nil)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
		t.Logf("Response body: %s", body)
	}

	// Парсим ответ
	var response struct {
		Notes  []models.Note `json:"notes"`
		Count  int           `json:"count"`
		Offset int           `json:"offset"`
		Limit  int           `json:"limit"`
	}

	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Count != 3 {
		t.Errorf("Expected count 3, got %d", response.Count)
	}

	if len(response.Notes) != 3 {
		t.Errorf("Expected 3 notes, got %d", len(response.Notes))
	}
}

// TestUpdateNote тестирует обновление заметки
func TestUpdateNote(t *testing.T) {
	helper := setupTestServer(t)

	// Создаем заметку
	createInput := models.CreateNoteInput{
		Title:   "Original Title",
		Content: "Original Content",
	}

	_, body := helper.makeRequest(t, "POST", "/api/notes", createInput)
	var created models.Note
	json.Unmarshal([]byte(body), &created)

	// Обновляем её
	updateInput := models.UpdateNoteInput{
		Title:   "Updated Title",
		Content: "Updated Content",
	}

	resp2, body2 := helper.makeRequest(
		t,
		"PUT",
		fmt.Sprintf("/api/notes/%d", created.ID),
		updateInput,
	)

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp2.StatusCode)
		t.Logf("Response body: %s", body2)
	}

	var updated models.Note
	json.Unmarshal([]byte(body2), &updated)

	if updated.Title != updateInput.Title {
		t.Errorf("Expected title %q, got %q", updateInput.Title, updated.Title)
	}

	if updated.Content != updateInput.Content {
		t.Errorf("Expected content %q, got %q", updateInput.Content, updated.Content)
	}

	// UpdatedAt должен измениться
	if updated.UpdatedAt == created.UpdatedAt {
		t.Error("Expected UpdatedAt to change")
	}
}

// TestDeleteNote тестирует удаление заметки
func TestDeleteNote(t *testing.T) {
	helper := setupTestServer(t)

	// Создаем заметку
	createInput := models.CreateNoteInput{
		Title:   "To Delete",
		Content: "This will be deleted",
	}

	_, body := helper.makeRequest(t, "POST", "/api/notes", createInput)
	var created models.Note
	json.Unmarshal([]byte(body), &created)

	// Удаляем её
	resp2, body2 := helper.makeRequest(
		t,
		"DELETE",
		fmt.Sprintf("/api/notes/%d", created.ID),
		nil,
	)

	if resp2.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp2.StatusCode)
		t.Logf("Response body: %s", body2)
	}

	// Проверяем, что она действительно удалена
	resp3, _ := helper.makeRequest(t, "GET", fmt.Sprintf("/api/notes/%d", created.ID), nil)
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("Expected deleted note to return 404, got %d", resp3.StatusCode)
	}
}

// TestHealthCheck тестирует health check endpoint
func TestHealthCheck(t *testing.T) {
	helper := setupTestServer(t)

	resp, body := helper.makeRequest(t, "GET", "/health", nil)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response map[string]string
	json.Unmarshal([]byte(body), &response)

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", response["status"])
	}
}

// TestConcurrentCreates тестирует конкурентное создание заметок
func TestConcurrentCreates(t *testing.T) {
	helper := setupTestServer(t)

	// Создаем 10 заметок конкурентно
	done := make(chan error, 10)

	for i := 1; i <= 10; i++ {
		go func(index int) {
			input := models.CreateNoteInput{
				Title:   fmt.Sprintf("Concurrent %d", index),
				Content: fmt.Sprintf("Content %d", index),
			}

			resp, _ := helper.makeRequest(t, "POST", "/api/notes", input)
			if resp.StatusCode != http.StatusCreated {
				done <- fmt.Errorf("failed to create note %d", index)
			} else {
				done <- nil
			}
		}(i)
	}

	// Проверяем результаты
	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent create failed: %v", err)
		}
	}

	// Проверяем, что все 10 записей в БД
	resp, body := helper.makeRequest(t, "GET", "/api/notes?limit=100", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get notes: %d", resp.StatusCode)
	}

	var response struct {
		Count int `json:"count"`
	}
	json.Unmarshal([]byte(body), &response)

	if response.Count != 10 {
		t.Errorf("Expected 10 notes, got %d", response.Count)
	}
}
