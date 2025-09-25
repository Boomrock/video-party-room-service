package database

import (
	"crypto/rand"
	"database/sql"
	"fmt"
)

func (db *DB) CreateRoomsTables() error {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT UNIQUE,
		video TEXT NULL
	);
	CREATE TABLE IF NOT EXISTS users_in_room (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		room_id INTEGER NOT NULL
	)`
	_, err := db.conn.Exec(createTablesSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблиц: %w", err)
	}
	fmt.Println("Таблица 'rooms' и 'users_in_room' готова.")
	return nil
}

type Room struct {
	ID    int            `json:"id"`
	Key   string         `json:"key"`
	Video sql.NullString `json:"video"`
	Users []User         `json:"users"`
}

// CreateRoom создает новую комнату
func (db *DB) CreateRoom() (*Room, error) {
	insertSQL := `INSERT INTO rooms (key) VALUES (?)`

	key := rand.Text() // Предполагается, что у вас есть такая функция
	row, err := db.conn.Exec(insertSQL, key)
	
	if err != nil {
		return nil, fmt.Errorf("ошибка вставки комнаты: %w", err)
	}
	roomID, err := row.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения id созданной комнаты: %w", err)
	}
	room := Room{
		ID:  int(roomID),
		Key: key,
	}

	fmt.Printf("Созданна комната: key - '%s'\n", key)
	return &room, nil
}

func (db *DB) GetRoomByID(id int) (*Room, error) {
	querySQL := `SELECT id, key, video FROM rooms WHERE id = ?`
	row := db.conn.QueryRow(querySQL, id)

	var room Room

	err := row.Scan(&room.ID, &room.Key, &room.Video)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения комнаты по ID: %w", err)
	}

	room.Users, _ = db.GetUsersInRoom(room.ID)

	return &room, nil
}

func (db *DB) GetUsersInRoom(roomID int) ([]User, error) {
	querySQL := `SELECT user_id FROM users_in_room WHERE room_id = ?`
	rows, err := db.conn.Query(querySQL, roomID)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var userID int
		err := rows.Scan(&userID)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}

		user, err := db.GetUserByID(userID)
		if err != nil {
			return nil, fmt.Errorf("ошибка поиска пользователя %d: %w", userID, err)
		}

		users = append(users, *user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	return users, nil
}

func (db *DB) AddUserInRoom(userID, roomID int) error {
	insertSQL := `INSERT INTO users_in_room (room_id, user_id) VALUES (?, ?)`
	_, err := db.conn.Exec(insertSQL, roomID, userID)
	if err != nil {
		return fmt.Errorf("ошибка добавления пользователя в комнату(room_id: '%d', user_id: '%d'): %w", roomID, userID, err)
	}
	fmt.Printf("Пользователь добавлен: пользователь: '%d' -> комната: '%d'\n", userID, roomID)
	return nil
}

// RemoveUserFromRoom удаляет пользователя из комнаты.
func (db *DB) RemoveUserFromRoom(userID, roomID int) error {
	result, err := db.conn.Exec(
		"DELETE FROM users_in_room WHERE user_id = ? AND room_id = ?",
		userID, roomID,
	)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя из комнаты: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("пользователь %d не найден в комнате %d", userID, roomID)
	}

	fmt.Printf("Пользователь %d удалён из комнаты %d\n", userID, roomID)
	return nil
}

// GetRoomByKey получает комнату по её уникальному ключу.
func (db *DB) GetRoomByKey(key string) (*Room, error) {
	querySQL := `SELECT id, key, video FROM rooms WHERE key = ?`
	row := db.conn.QueryRow(querySQL, key)

	var room Room

	err := row.Scan(&room.ID, &room.Key, &room.Video)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения комнаты по ключу: %w", err)
	}

	// Загружаем пользователей в комнате
	room.Users, _ = db.GetUsersInRoom(room.ID)

	return &room, nil
}

// SetRoomVideo устанавливает видео для комнаты.
func (db *DB) SetRoomVideo(roomID int, video string) error {
	var query string
	query = "UPDATE rooms SET video = ? WHERE id = ?"

	result, err := db.conn.Exec(query, video, roomID)
	if err != nil {
		return fmt.Errorf("ошибка установки видео для комнаты: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("комната с ID %d не найдена", roomID)
	}

	fmt.Printf("Видео %s установлено для комнаты %d\n", video, roomID)
	return nil
}
func (db *DB) SetRoomVideoByKey(key string, video string) error {
	
	room, err := db.GetRoomByKey(key)
	if err != nil {
		return fmt.Errorf("ошибка поиска комнаты в установке видео для комнаты: %w", err)
	}

	var query string

	query = "UPDATE rooms SET video = ? WHERE id = ?"

	result, err := db.conn.Exec(query, video, room.ID)
	if err != nil {
		return fmt.Errorf("ошибка установки видео для комнаты: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("комната с ID %d не найдена", room.ID)
	}

	fmt.Printf("Видео %s установлено для комнаты %d\n", video, room.ID)
	return nil
}

// Вспомогательная функция для удаления дубликатов комнат
func (db *DB) removeDuplicateRooms(rooms []Room) []Room {
	seen := make(map[int]bool)
	result := []Room{}

	for _, room := range rooms {
		if !seen[room.ID] {
			seen[room.ID] = true
			result = append(result, room)
		}
	}

	return result
}
