package database

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/MSTimX/Snowops-roles/internal/models"
)

// DB хранит глобальное подключение к базе данных.
var DB *gorm.DB

// Init загружает конфигурацию и инициализирует подключение к PostgreSQL.
func Init() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("не удалось загрузить .env файл: %v", err)
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatalf("переменная окружения DB_DSN не установлена")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("не удалось подключиться к базе данных: %v", err)
	}

	DB = db
}

// Migrate выполняет авто-миграции для всех моделей.
func Migrate() {
	if DB == nil {
		log.Fatalf("подключение к базе данных не инициализировано")
	}

	if err := DB.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Driver{},
		&models.Vehicle{},
	); err != nil {
		log.Fatalf("ошибка авто-миграции: %v", err)
	}
}
