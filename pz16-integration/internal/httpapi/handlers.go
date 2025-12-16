package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"example.com/pz16-integration/internal/models"
	"example.com/pz16-integration/internal/service"
)

// Router — маршрутизатор для HTTP API
type Router struct {
	Service *service.Service
}

// Register регистрирует все маршруты в Gin engine
func (rt Router) Register(engine *gin.Engine) {
	// CRUD операции
	engine.POST("/api/notes", rt.createNote)
	engine.GET("/api/notes/:id", rt.getNote)
	engine.GET("/api/notes", rt.getNotes)
	engine.PUT("/api/notes/:id", rt.updateNote)
	engine.DELETE("/api/notes/:id", rt.deleteNote)

	// Health check
	engine.GET("/health", rt.health)
}

// health возвращает статус приложения
func (rt Router) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// createNote создает новую заметку
// POST /api/notes
// Body: {"title": "...", "content": "..."}
func (rt Router) createNote(c *gin.Context) {
	var input models.CreateNoteInput

	// Парсим JSON из body
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	// Вызываем сервис
	note, err := rt.Service.CreateNote(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Возвращаем созданную заметку со статусом 201
	c.JSON(http.StatusCreated, note)
}

// getNote получает заметку по ID
// GET /api/notes/:id
func (rt Router) getNote(c *gin.Context) {
	// Парсим ID из URL
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid note ID",
		})
		return
	}

	// Вызываем сервис
	note, err := rt.Service.GetNote(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, note)
}

// getNotes получает список заметок с пагинацией
// GET /api/notes?offset=0&limit=10
func (rt Router) getNotes(c *gin.Context) {
	// Парсим query параметры
	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Вызываем сервис
	notes, err := rt.Service.GetNotes(c.Request.Context(), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Если список пуст, возвращаем пустой слайс, а не null
	if notes == nil {
		notes = make([]models.Note, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"notes":  notes,
		"count":  len(notes),
		"offset": offset,
		"limit":  limit,
	})
}

// updateNote обновляет заметку
// PUT /api/notes/:id
// Body: {"title": "...", "content": "..."}
func (rt Router) updateNote(c *gin.Context) {
	// Парсим ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid note ID",
		})
		return
	}

	var input models.UpdateNoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	// Вызываем сервис
	note, err := rt.Service.UpdateNote(c.Request.Context(), id, input)
	if err != nil {
		if errors.Is(err, errors.New("note not found")) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, note)
}

// deleteNote удаляет заметку
// DELETE /api/notes/:id
func (rt Router) deleteNote(c *gin.Context) {
	// Парсим ID
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid note ID",
		})
		return
	}

	// Вызываем сервис
	if err := rt.Service.DeleteNote(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Возвращаем 204 No Content (нет тела ответа)
	c.Status(http.StatusNoContent)
}
