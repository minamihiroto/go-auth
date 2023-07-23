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
		log.Fatal("Error loading .env file")
	}

	userService := user.NewService("myapp.db")

	r := mux.NewRouter()

	s := user.NewService("myapp.db")
	r.HandleFunc("/register", s.RegisterHandler)
	r.HandleFunc("/login", s.LoginHandler)
	r.HandleFunc("/logout", s.LogoutHandler)
	r.HandleFunc("/auth", userService.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is an authenticated response"))
	}))

	http.ListenAndServe(":8080", r)
}
