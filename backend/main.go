package main

import (
	"log"
	"os"
	
	"food-recipes-backend/config"
	"food-recipes-backend/handlers"
	"food-recipes-backend/middleware"
	"food-recipes-backend/models"
	
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}
	
	cfg := config.Load()
	
	// Initialize database
	dsn := cfg.DatabaseURL
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	// Auto migrate tables
	if err := db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Recipe{},
		&models.Ingredient{},
		&models.Step{},
		&models.RecipeImage{},
		&models.Like{},
		&models.Bookmark{},
		&models.Comment{},
		&models.Rating{},
		&models.Purchase{},
	); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	
	// Create default categories
	createDefaultCategories(db)
	
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	recipeHandler := handlers.NewRecipeHandler(db)
	categoryHandler := handlers.NewCategoryHandler(db)
	uploadHandler := handlers.NewUploadHandler(cfg.UploadDir)
	paymentHandler := handlers.NewChapaPaymentHandler(db, cfg.ChapaSecretKey)
	
	// Setup Gin router
	router := gin.Default()
	
	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	// Serve uploaded files
	router.Static("/uploads", cfg.UploadDir)
	
	// Public routes
	public := router.Group("/api")
	{
		public.POST("/auth/signup", authHandler.Signup)
		public.POST("/auth/login", authHandler.Login)
		public.GET("/categories", categoryHandler.GetCategories)
		public.GET("/categories/:id/recipes", categoryHandler.GetCategoryRecipes)
		public.GET("/recipes", recipeHandler.GetRecipes)
		public.GET("/recipes/:id", middleware.OptionalAuthMiddleware(), recipeHandler.GetRecipe)
		public.POST("/upload", uploadHandler.UploadImage)
	}
	
	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// User routes
		protected.GET("/auth/profile", authHandler.GetProfile)
		
		// Recipe routes
		protected.POST("/recipes", recipeHandler.CreateRecipe)
		protected.PUT("/recipes/:id", recipeHandler.UpdateRecipe)
		protected.DELETE("/recipes/:id", recipeHandler.DeleteRecipe)
		protected.POST("/recipes/:id/like", recipeHandler.ToggleLike)
		protected.POST("/recipes/:id/bookmark", recipeHandler.ToggleBookmark)
		protected.POST("/recipes/:id/rating", recipeHandler.AddRating)
		protected.POST("/recipes/:id/comment", recipeHandler.AddComment)
		
		// Payment routes
		protected.POST("/payment/initialize", paymentHandler.InitializePayment)
		protected.GET("/payment/purchases", paymentHandler.GetUserPurchases)
	}
	
	// Payment verification (public callback)
	router.GET("/api/payment/verify", paymentHandler.VerifyPayment)
	
	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(router.Run(":" + cfg.Port))
}

func createDefaultCategories(db *gorm.DB) {
	categories := []models.Category{
		{Name: "Breakfast", Description: "Start your day right"},
		{Name: "Lunch", Description: "Midday meals"},
		{Name: "Dinner", Description: "Evening delights"},
		{Name: "Desserts", Description: "Sweet treats"},
		{Name: "Appetizers", Description: "Starters and snacks"},
		{Name: "Vegetarian", Description: "Plant-based recipes"},
		{Name: "Vegan", Description: "100% plant-based"},
		{Name: "Gluten-Free", Description: "No gluten ingredients"},
		{Name: "Quick & Easy", Description: "30 minutes or less"},
		{Name: "Healthy", Description: "Nutritious options"},
	}
	
	for _, category := range categories {
		var existing models.Category
		if err := db.Where("name = ?", category.Name).First(&existing).Error; err != nil {
			db.Create(&category)
		}
	}
}