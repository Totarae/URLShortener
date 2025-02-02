package main

import (
	"fmt"
	"github.com/Totarae/URLShortener/internal/handlers"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func main() {

	r := chi.NewRouter()
	r.Post("/", handlers.ReceiveURL)
	r.Get("/{id}", handlers.ResponseURL)

	fmt.Println("Сервер запущен на порту 8080...")
	http.ListenAndServe(":8080", r)

}
