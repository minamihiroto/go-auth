package user

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/dgrijalva/jwt-go/request"
	"github.com/go-redis/redis/v8"
)

const (
	mySigningKey = "secret"
)

type Service struct {
	db    *sql.DB
	redis *redis.Client
}

func NewService(dbFile string) *Service {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		panic(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	statement, _ := db.Prepare("CREATE TABLE IF NOT EXISTS user (username TEXT PRIMARY KEY, password TEXT)")
	statement.Exec()

	return &Service{db: db, redis: redisClient}
}

func (s *Service) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	hash, _ := hashPassword(password)

	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, "Could not complete request", http.StatusInternalServerError)
		return
	}

	stmt, err := tx.Prepare("INSERT INTO user(username, password) values(?, ?)")
	if err != nil {
		http.Error(w, "Could not complete request", http.StatusInternalServerError)
		return
	}

	_, err = stmt.Exec(username, hash)
	if err != nil {
		http.Error(w, "Could not complete request", http.StatusInternalServerError)
		return
	}

	tx.Commit()

	fmt.Fprintf(w, "User %s registered", username)
}

func (s *Service) LoginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	var hash string
	err := s.db.QueryRow("SELECT password FROM user WHERE username=?", username).Scan(&hash)
	if err != nil {
		http.Error(w, "Could not complete request", http.StatusInternalServerError)
		return
	}

	if !checkPasswordHash(password, hash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	tokenString, _ := generateJwt(username)
	fmt.Fprintf(w, "Token: %s", tokenString)
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateJwt(username string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = username
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	tokenString, err := token.SignedString([]byte(mySigningKey))
	return tokenString, err
}

func (s *Service) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(mySigningKey), nil
			})

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if token.Valid {
			tokenString := token.Raw
			_, err := s.redis.Get(context.Background(), tokenString).Result()

			if err != redis.Nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
		}
	})
}

func (s *Service) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(mySigningKey), nil
		})

	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if token.Valid {
		tokenString := token.Raw

		err := s.redis.Set(context.Background(), tokenString, tokenString, time.Hour).Err()
		if err != nil {
			http.Error(w, "Logout failed", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
	}

	fmt.Fprintf(w, "Successfully logged out")
}
