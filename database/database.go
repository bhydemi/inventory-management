package database

import (
	"log"
	"os"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"inventory_management/models"
)

// DB is the shared GORM handle used by every handler.
var DB *gorm.DB

// defaultDSN connects to a local PostgreSQL server using the standard
// port and the conventional "postgres" superuser, against a database
// named "inventory". Override it by setting the DATABASE_URL environment
// variable, e.g. if your local Postgres runs on a non-default port.
const defaultDSN = "host=localhost user=postgres password=postgres dbname=inventory port=5432 sslmode=disable"

// Connect opens the database connection, runs the schema migration, and
// seeds the table with starter data if it's empty.
func Connect() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&models.Item{}); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	DB = db
	seed()
}

// seed inserts a starter inventory the first time the server runs against
// an empty table. It's idempotent: once the table has data, it's a no-op.
func seed() {
	var count int64
	DB.Model(&models.Item{}).Count(&count)
	if count > 0 {
		log.Printf("database already has %d item(s), skipping seed", count)
		return
	}

	items := []models.Item{
		{ID: uuid.NewString(), Name: "Laptop", Stock: 10, Price: 999.99},
		{ID: uuid.NewString(), Name: "Smartphone", Stock: 20, Price: 699.99},
		{ID: uuid.NewString(), Name: "Headphones", Stock: 15, Price: 199.99},
		{ID: uuid.NewString(), Name: "Keyboard", Stock: 25, Price: 89.99},
		{ID: uuid.NewString(), Name: "Mouse", Stock: 30, Price: 49.99},
		{ID: uuid.NewString(), Name: "Monitor", Stock: 12, Price: 299.99},
		{ID: uuid.NewString(), Name: "Webcam", Stock: 18, Price: 79.99},
		{ID: uuid.NewString(), Name: "Printer", Stock: 7, Price: 149.99},
		{ID: uuid.NewString(), Name: "Tablet", Stock: 5, Price: 399.99},
		{ID: uuid.NewString(), Name: "Smartwatch", Stock: 14, Price: 249.99},
	}

	if err := DB.Create(&items).Error; err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}
	log.Printf("seeded database with %d item(s)", len(items))
}
