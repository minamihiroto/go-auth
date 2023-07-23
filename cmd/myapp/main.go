package main

import (
	"log"
	"myapp/internal/user"
	"net/http"

	"github.com/joho/godotenv"

	"github.com/gorilla/mux"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	userService, err := user.NewService("myapp.db")
	if err != nil {
		log.Fatalf("Error creating user service: %v", err)
	}

	r := mux.NewRouter()

	r.HandleFunc("/register", userService.RegisterHandler)
	r.HandleFunc("/login", userService.LoginHandler)
	r.HandleFunc("/logout", userService.LogoutHandler)
	r.HandleFunc("/auth", userService.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is an authenticated response"))
	}))

	http.ListenAndServe(":8080", r)
}
