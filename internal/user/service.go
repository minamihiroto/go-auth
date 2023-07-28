package user

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
	_ "github.com/mattn/go-sqlite3"
)

type Service struct {
	db           *sql.DB
	redis        *redis.Client
	mySigningKey string
}

func NewService(dbFile string, mySigningKey string) (*Service, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Printf("Error opening database: %v", err)
		return nil, err
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	statement, err := db.Prepare("CREATE TABLE IF NOT EXISTS user (email TEXT PRIMARY KEY, password TEXT)")
	if err != nil {
		log.Printf("Error preparing database statement: %v", err)
		return nil, err
	}

	_, err = statement.Exec()
	if err != nil {
		log.Printf("Error executing database statement: %v", err)
		return nil, err
	}

	return &Service{db: db, redis: redisClient, mySigningKey: mySigningKey}, nil
}

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

func hashPassword(password string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	return hex.EncodeToString(hasher.Sum(nil))
}

func checkPasswordHash(password, hash string) bool {
	passwordHash := hashPassword(password)
	return passwordHash == hash
}

func (s *Service) generateJwt(email string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["email"] = email
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	tokenString, err := token.SignedString([]byte(s.mySigningKey))
	return tokenString, err
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

		token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.mySigningKey), nil
		})
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			log.Printf("Invalid token: %v", err)
			return
		}

		if token.Valid {
			_, err := s.redis.Get(context.Background(), bearerToken).Result()
			if err != redis.Nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				log.Printf("Invalid token: %v", err)
				return
			}

			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			log.Printf("Invalid token: %v", err)
		}
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
