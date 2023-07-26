package main

import (
	"log"
	"net/http"
	"os"

	"myapp/internal/user"

	"github.com/gorilla/mux"
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

	r := mux.NewRouter()

	r.HandleFunc("/register", userService.RegisterHandler)
	r.HandleFunc("/login", userService.LoginHandler)
	r.HandleFunc("/logout", userService.LogoutHandler)
	r.HandleFunc("/auth", userService.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("This is an authenticated response"))
	}))

	http.ListenAndServe(":8080", r)
}
