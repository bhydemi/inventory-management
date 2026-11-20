package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"inventory_management/database"
	"inventory_management/handlers"
	"inventory_management/middleware"
)

func main() {
	database.Connect()

	router := gin.Default()
	router.Use(middleware.RateLimiter())

	inventory := router.Group("/inventory")
	{
		inventory.GET("", handlers.GetItems)
		inventory.GET("/:id", handlers.GetItem)
		inventory.POST("", handlers.CreateItem)
		inventory.PUT("/:id", handlers.UpdateItem)
		inventory.DELETE("/:id", handlers.DeleteItem)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
