package models

// Note представляет заметку в системе
type Note struct {
	// ID — первичный ключ, автоинкремент в БД
	ID int64 `db:"id" json:"id"`
	
	// Title — заголовок заметки
	Title string `db:"title" json:"title"`
	
	// Content — содержимое заметки
	Content string `db:"content" json:"content"`
	
	// CreatedAt — время создания (устанавливается БД автоматически)
	CreatedAt string `db:"created_at" json:"created_at"`
	
	// UpdatedAt — время последнего обновления (обновляется триггером БД)
	UpdatedAt string `db:"updated_at" json:"updated_at"`
}

// CreateInput — input для создания заметки (без ID и временных меток)
type CreateNoteInput struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// UpdateInput — input для обновления заметки
type UpdateNoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
