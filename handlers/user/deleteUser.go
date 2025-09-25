package user

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"room/database"
	"strconv"
)

// deleteUserResponse — структура для ответа при успешном удалении
type deleteUserResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	ID      int    `json:"id"`
}

func DeleteUser(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			slog.Error("Отсутствует обязательный параметр: id",
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Missing required parameter: id", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			slog.Error("Некорректное значение параметра id",
				"значение", idStr,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}

		if id <= 0 {
			slog.Error("ID должно быть положительным числом",
				"id", id,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Invalid id: must be positive", http.StatusBadRequest)
			return
		}

		err = db.DeleteUser(id)
		if err != nil {
			slog.Error("Не удалось удалить пользователя",
				"error", err,
				"id", id,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		// Успешный ответ
		response := deleteUserResponse{
			Status:  "success",
			Message: "User deleted successfully",
			ID:      id,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("Не удалось закодировать ответ",
				"error", err,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			// Запись уже начата, ничего больше сделать нельзя
			return
		}
	}
}