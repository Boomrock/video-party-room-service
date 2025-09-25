package user

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"room/database"
)

// createUserResponse — структура для ответа при успешном удалении
type createUserResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	User    database.User `json:"user"`
}

func CreateUser(database *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			slog.Error("Отсутствует обязательный параметр: name",
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Missing required parameter: name", http.StatusBadRequest)
			return
		}
		user, err := database.CreateUser(name)
		if err != nil {
			slog.Error("Не удалось создать пользователя",
				"error", err,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "User not created", http.StatusInternalServerError)
			return
		}

		// Устанавливаем тип содержимого
		w.Header().Set("Content-Type", "application/json")
		response := createUserResponse{
			Status:  "success",
			Message: "User create successfully",
			User:    *user,
		}

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			slog.Error("Ошибка при отправке ответа с пользователем",
				"error", err,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			// Нельзя вызвать http.Error после начала записи в w
			return
		}
	}
}
