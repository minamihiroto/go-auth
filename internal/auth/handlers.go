package auth

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func (s *Service) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	hash := hashPassword(password)

	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, "Could not begin transaction", http.StatusInternalServerError)
		log.Printf("Could not begin transaction: %v", err)
		return
	}

	stmt, err := tx.Prepare("INSERT INTO user(email, password) values(?, ?)")
	if err != nil {
		http.Error(w, "Could not prepare statement", http.StatusInternalServerError)
		log.Printf("Could not prepare statement: %v", err)
		return
	}

	_, err = stmt.Exec(email, hash)
	if err != nil {
		http.Error(w, "Could not execute statement", http.StatusInternalServerError)
		log.Printf("Could not execute statement: %v", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "Could not commit transaction", http.StatusInternalServerError)
		log.Printf("Could not commit transaction: %v", err)
		return
	}

	message := "User " + email + " registered"
	w.Write([]byte(message))
	log.Print(message)
}

func (s *Service) LoginHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	var hash string
	err := s.db.QueryRow("SELECT password FROM user WHERE email=?", email).Scan(&hash)
	if err != nil {
		http.Error(w, "Could not query user password", http.StatusInternalServerError)
		log.Printf("Could not query user password: %v", err)
		return
	}

	if !checkPasswordHash(password, hash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	tokenString, err := s.generateJwt(email)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		log.Printf("Could not generate token: %v", err)
		return
	}
	message := "Token: " + tokenString
	w.Write([]byte(message))
	log.Print(message)
}

func (s *Service) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			log.Print("Invalid token: no Authorization header")
			return
		}

		bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
		if bearerToken == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			log.Print("Invalid token: no Bearer token")
			return
		}

		token, _ := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				errMsg := "unexpected signing method: " + token.Header["alg"].(string)
				http.Error(w, errMsg, http.StatusUnauthorized)
				log.Printf("Invalid token: %s", errMsg)
				return nil, nil
			}
			return []byte(s.mySigningKey), nil
		})

		if token == nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			log.Print("Invalid token: parsing error or invalid token")
			return
		}

		_, err := s.redis.Get(context.Background(), bearerToken).Result()
		if err != redis.Nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			log.Printf("Invalid token: %v", err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Service) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Could not set token in redis", http.StatusInternalServerError)
		log.Print("Could not set token in redis: no Authorization header")
		return
	}

	bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
	if bearerToken == "" {
		http.Error(w, "Could not set token in redis", http.StatusInternalServerError)
		log.Print("Could not set token in redis: no Bearer token")
		return
	}

	err := s.redis.Set(context.Background(), bearerToken, bearerToken, time.Hour).Err()
	if err != nil {
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		log.Printf("Logout failed: %v", err)
		return
	}

	message := "Successfully logged out"
	w.Write([]byte(message))
	log.Print(message)
}
