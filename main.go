package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gaanon/gorecipes_v2/config" 
	"github.com/gaanon/gorecipes_v2/handlers"
	"github.com/gaanon/gorecipes_v2/store"  
)

func main() {
	// Load configuration
	dbCfg := config.DefaultDBConfig()
	// Production reminder: Load credentials securely, e.g., from environment variables or a config file.
	// Ensure config.go has your actual DB credentials if you haven't updated it yet.

	// Initialize database pool
	dbPool, err := store.NewDBPool(dbCfg)
	if err != nil {
		log.Fatalf("Failed to initialize database connection: %v", err)
	}
	defer dbPool.Close() // Ensure the pool is closed when the application exits

	// Initialize store
	recipeStore := store.NewRecipeStore(dbPool)

	// Initialize handlers
	recipeHandler := handlers.NewRecipeHandler(recipeStore)

	// Initialize Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Recipe routes
	apiV1 := router.Group("/api/v1") // Group routes under /api/v1
	{
		recipesGroup := apiV1.Group("/recipes")
		{
			recipesGroup.POST("", recipeHandler.CreateRecipe)
			recipesGroup.GET("", recipeHandler.ListRecipes)
			recipesGroup.GET("/:id", recipeHandler.GetRecipe)
			recipesGroup.PUT("/:id", recipeHandler.UpdateRecipe)
			recipesGroup.DELETE("/:id", recipeHandler.DeleteRecipe)
		}
	}

	// Start the server
	serverAddr := ":8080" // Make this configurable later
	log.Printf("Server starting on %s", serverAddr)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
