// Package main wires up the inventory management API.
//
// @title           Inventory Management API
// @version         1.0
// @description     A RESTful inventory management service with pagination, filtering, sorting, rate limiting, caching, and JWT-protected writes.
// @BasePath        /
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT returned from /login.
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "inventory_management/docs"

	"inventory_management/database"
	"inventory_management/handlers"
	"inventory_management/middleware"
)

func main() {
	database.Connect()

	router := gin.Default()

	// Swagger's own page loads several static assets (JS, CSS, favicons)
	// in one burst -- rate limiting is for the API, not its docs, so it's
	// scoped to the API routes below rather than applied globally.
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/")
	api.Use(middleware.RateLimiter())

	api.POST("/login", handlers.Login)

	inventory := api.Group("/inventory")
	{
		// Reads are open to anyone.
		inventory.GET("", handlers.GetItems)
		inventory.GET("/:id", handlers.GetItem)

		// Writes require a valid JWT (see POST /login).
		inventory.POST("", middleware.RequireAuth(), handlers.CreateItem)
		inventory.PUT("/:id", middleware.RequireAuth(), handlers.UpdateItem)
		inventory.DELETE("/:id", middleware.RequireAuth(), handlers.DeleteItem)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
