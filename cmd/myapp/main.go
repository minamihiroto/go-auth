package main

import (
	"log"
	"net/http"
	"os"

	"myapp/internal/auth"
)

func main() {
	mySigningKey := os.Getenv("MY_SIGNING_KEY")
	if mySigningKey == "" {
		log.Fatalf("Environment variable MY_SIGNING_KEY is not set")
	}

	userService, err := user.NewService("myapp.db", mySigningKey)
	if err != nil {
		log.Fatalf("Error creating user service: %v", err)
	}

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		userService.RegisterHandler(w, r)
	})
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		userService.LoginHandler(w, r)
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		userService.LogoutHandler(w, r)
	})
	http.HandleFunc("/auth", userService.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is an authenticated response"))
	}))

	http.ListenAndServe(":8080", nil)
}
