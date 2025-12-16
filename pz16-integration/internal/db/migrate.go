package db

import "database/sql"

func MustApplyMigrations(db *sql.DB) {
  // Простейшая миграция без внешних инструментов:
  _, err := db.Exec(`
    CREATE TABLE IF NOT EXISTS notes (
      id BIGSERIAL PRIMARY KEY,
      title TEXT NOT NULL,
      content TEXT NOT NULL,
      created_at TIMESTAMP NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMP NOT NULL DEFAULT NOW()
    );
    CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
    BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$ LANGUAGE plpgsql;
    DROP TRIGGER IF EXISTS trg_notes_updated ON notes;
    CREATE TRIGGER trg_notes_updated BEFORE UPDATE ON notes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
  `)
  if err != nil { panic(err) }
}
