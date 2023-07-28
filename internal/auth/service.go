package auth

import (
	"database/sql"
	"log"

	"github.com/go-redis/redis/v8"
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
