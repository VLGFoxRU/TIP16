package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"example.com/pz16-integration/internal/db"
	"example.com/pz16-integration/internal/httpapi"
	"example.com/pz16-integration/internal/repo"
	"example.com/pz16-integration/internal/service"
)

func main() {
	// Читаем DSN (connection string) из переменной окружения
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is not set")
	}

	// Подключаемся к БД
	dbx, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer dbx.Close()

	// Проверяем подключение
	if err := dbx.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	log.Println("✓ Connected to database")

	// Применяем миграции
	db.MustApplyMigrations(dbx)
	log.Println("✓ Migrations applied")

	// Создаем слои приложения
	noteRepo := repo.NoteRepo{DB: dbx}
	svc := service.Service{Notes: noteRepo}

	// Настраиваем Gin
	engine := gin.Default()

	// Регистрируем маршруты
	router := httpapi.Router{Service: &svc}
	router.Register(engine)

	// Запускаем сервер
	addr := ":8080"
	log.Printf("Starting server on %s", addr)

	if err := http.ListenAndServe(addr, engine); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
