package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName        string
	AppPort        string
	AppEnv         string
	JWTSecret      string
	GoogleClientID string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system env")
	}

	return &Config{
		AppName:        os.Getenv("APP_NAME"),
		AppPort:        os.Getenv("APP_PORT"),
		AppEnv:         os.Getenv("APP_ENV"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),

		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
	}
}

func (c *Config) DBDsn() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s ",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort,
	)
}

//port=%s sslmode=require TimeZone=Asia/Jakarta prepare_threshold=0
