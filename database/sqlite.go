// Package db предоставляет функции для работы с базой данных SQLite.
package database

import (
	"database/sql"
	"fmt"

	// Импортируем драйвер SQLite. Пустой импорт _ регистрирует драйвер.
	_ "github.com/mattn/go-sqlite3"
)

// DB - структура, инкапсулирующая соединение с базой данных.
type DB struct {
	conn *sql.DB
}

// New создает новое подключение к базе данных SQLite.
// filepath - путь к файлу базы данных (например, "./example.db").
// Возвращает указатель на DB и ошибку, если подключение не удалось.
func New(filepath string) (*DB, error) {
	// Открываем соединение с базой данных
	dbConn, err := sql.Open("sqlite3", filepath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть базу данных: %w", err)
	}

	// Проверяем соединение
	if err := dbConn.Ping(); err != nil {
		// Важно закрыть соединение, если пинг не удался
		dbConn.Close()
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	fmt.Println("Успешное подключение к SQLite!")
	return &DB{conn: dbConn}, nil
}
func (db *DB) CreateTable() error {
	if err := db.CreateUsersTable(); err != nil {
		return fmt.Errorf("ошибка users: %w", err)
	}
	if err := db.CreateRoomsTables(); err != nil {
		return fmt.Errorf("ошибка rooms: %w", err)
	}
	return nil
}

// Close закрывает соединение с базой данных.
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}
