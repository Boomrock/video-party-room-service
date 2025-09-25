package room

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"room/database"
)

func SetVideo(database *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			slog.Error("Отсутствует обязательный параметр: key",
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Missing required parameter: key", http.StatusBadRequest)
			return
		}
		file_name := r.URL.Query().Get("file_name")

		if file_name == "" {
			slog.Error("Отсутствует обязательный параметр: file_name",
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Missing required parameter: file_name", http.StatusBadRequest)
			return
		}
		err := database.SetRoomVideoByKey(key, file_name)
		if err != nil {
			slog.Error(fmt.Sprintf("Не удалось установить видео для комнаты комнату с Key: %s", key),
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
				"ошибка", err.Error(),
			)
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
	
		// Устанавливаем тип содержимого
		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(file_name)
		if err != nil {
			slog.Error("Ошибка при отправке ответа с ключом комнаты",
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
