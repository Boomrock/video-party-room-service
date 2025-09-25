package room

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"room/database"
)

// createRoomResponse — структура для ответа при успешном удалении
type createRoomResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Room    database.Room `json:"room"`
}

func CreateRoom(database *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		room, err := database.CreateRoom()
		if err != nil {
			slog.Error("Не удалось создать комнату",
				"error", err,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Room not created", http.StatusInternalServerError)
			return
		}

		// Устанавливаем тип содержимого
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		response := createRoomResponse{
			Status:  "success",
			Message: "Room created successfully",
			Room:    *room,
		}

		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			slog.Error("Ошибка при отправке ответа с ключом комнаты",
				"error", err,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			return
		}
	}
}
