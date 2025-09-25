package main

import (
	"fmt"
	"net/http"
	"room/database"
	"room/handlers/room"
	"room/handlers/user"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

// 1. Создание комнаты
// 2. Закачка на сервер
// 3. Синхронизация видео

func main() {
	sqllite, err := database.New("./sqlite.db")
	if err != nil {
		fmt.Printf(fmt.Errorf("база данных не открылась: %w", err).Error())
		return
	}
	err = sqllite.CreateTable()
	if err != nil {
		fmt.Printf(fmt.Errorf("база данных не создалась: %w", err).Error())
		return
	}

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)                 // Восстановление после паники
	router.Use(middleware.Timeout(30 * time.Second)) // Таймаут на обработку

	router.Route("/room", func(r chi.Router) {
		r.Get("/", room.GetRoom(sqllite))
		r.Get("/create", room.CreateRoom(sqllite))
		r.Get("/ws", room.VideoController())
	})
	router.Route("/user", func(r chi.Router) {
		r.Get("/create", user.CreateUser(sqllite))
		r.Get("/delete", user.DeleteUser(sqllite))
	})

	fmt.Println("Сервер запущен на http://localhost:8080")
	fmt.Println("Открой в браузере: http://localhost:8080/video")
	http.ListenAndServe(":8080", router)
}
