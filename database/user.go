package database

import "fmt"

func (db *DB) CreateUsersTable() error {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);
	`
	_, err := db.conn.Exec(createTablesSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблиц: %w", err)
	}
	fmt.Println("Таблица 'users' готова.")
	return nil
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (db *DB) GetUserByID(id int) (*User, error) {
	querySQL := `SELECT id, name FROM users WHERE id == ?`
	row := db.conn.QueryRow(querySQL, id)

	var u User
	err := row.Scan(&u.ID, &u.Name)
	if err != nil {

		return nil, fmt.Errorf("ошибка получения пользователя по ID: %w", err)
	}

	// Видео найдено
	return &u, nil
}

// CreateUser добавляет нового пользователя в базу данных.
// Возвращает указатель на созданного пользователя и ошибку, если имя уже занято.
func (db *DB) CreateUser(name string) (*User, error) {
	// Проверяем, существует ли уже пользователь с таким именем
	existingUser, _ := db.GetUserByName(name)
	if existingUser != nil {
		return nil, fmt.Errorf("пользователь с именем '%s' уже существует", name)
	}

	insertSQL := `INSERT INTO users (name) VALUES (?)`
	result, err := db.conn.Exec(insertSQL, name)
	if err != nil {
		return nil, fmt.Errorf("ошибка добавления пользователя: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения ID нового пользователя: %w", err)
	}

	user := &User{
		ID:   int(id),
		Name: name,
	}

	fmt.Printf("Пользователь добавлен: ID='%d', Имя='%s'\n", id, name)
	return user, nil
}

// GetAllUsers получает всех пользователей из базы данных.
func (db *DB) GetAllUsers() ([]User, error) {
	querySQL := `SELECT id, name FROM users`
	rows, err := db.conn.Query(querySQL)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Name)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	return users, nil
}

// GetUserByName получает пользователя по его имени.
// Возвращает nil, если пользователь не найден (без ошибки).
func (db *DB) GetUserByName(name string) (*User, error) {
	querySQL := `SELECT id, name FROM users WHERE name = ?`
	row := db.conn.QueryRow(querySQL, name)

	var u User
	err := row.Scan(&u.ID, &u.Name)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя по имени: %w", err)
	}

	return &u, nil
}

// UpdateUser обновляет имя пользователя по его ID.
func (db *DB) UpdateUser(id int, newName string) error {
	// Проверяем, существует ли пользователь
	_, err := db.GetUserByID(id)
	if err != nil {
		return fmt.Errorf("пользователь с ID %d не найден: %w", id, err)
	}

	// Проверяем, не существует ли уже пользователя с таким именем
	existingUser, _ := db.GetUserByName(newName)
	if existingUser != nil && existingUser.ID != id {
		return fmt.Errorf("пользователь с именем '%s' уже существует", newName)
	}

	updateSQL := `UPDATE users SET name = ? WHERE id = ?`
	result, err := db.conn.Exec(updateSQL, newName, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с ID %d не найден", id)
	}

	fmt.Printf("Пользователь с ID %d обновлен: новое имя='%s'\n", id, newName)
	return nil
}

// DeleteUser удаляет пользователя по ID.
// Сначала удаляет пользователя из всех комнат, затем удаляет самого пользователя.
func (db *DB) DeleteUser(id int) error {
	// Проверяем, существует ли пользователь
	_, err := db.GetUserByID(id)
	if err != nil {
		return fmt.Errorf("пользователь с ID %d не найден: %w", id, err)
	}

	// Удаляем пользователя из всех комнат
	_, err = db.conn.Exec("DELETE FROM users_in_room WHERE user_id = ?", id)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя из комнат: %w", err)
	}

	// Удаляем комнаты, где пользователь является владельцем
	_, err = db.conn.Exec("DELETE FROM rooms WHERE owner = ?", id)
	if err != nil {
		return fmt.Errorf("ошибка удаления комнат владельца: %w", err)
	}

	// Удаляем самого пользователя
	result, err := db.conn.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с ID %d не найден", id)
	}

	fmt.Printf("Пользователь с ID %d удален\n", id)
	return nil
}

// UserExists проверяет, существует ли пользователь с указанным ID.
func (db *DB) UserExists(id int) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки существования пользователя: %w", err)
	}
	return count > 0, nil
}

// IsUserInRoom проверяет, находится ли пользователь в указанной комнате.
func (db *DB) IsUserInRoom(userID, roomID int) (bool, error) {
	var count int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM users_in_room WHERE user_id = ? AND room_id = ?",
		userID, roomID,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("ошибка проверки участия в комнате: %w", err)
	}

	return count > 0, nil
}
