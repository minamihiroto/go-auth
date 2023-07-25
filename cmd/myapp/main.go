package main

import (
	"log"
	"net/http"
	"os"

	"myapp/internal/user"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	mySigningKey := os.Getenv("MY_SIGNING_KEY")
	userService, err := user.NewService("myapp.db", mySigningKey)
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
