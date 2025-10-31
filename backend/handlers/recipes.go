package handlers

import (
	"net/http"
	"strconv"
	
	"food-recipes-backend/models"
	"food-recipes-backend/utils"
	
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RecipeHandler struct {
	DB *gorm.DB
}

func NewRecipeHandler(db *gorm.DB) *RecipeHandler {
	return &RecipeHandler{DB: db}
}

func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	var recipeInput struct {
		Title            string                   `json:"title" binding:"required"`
		Description      string                   `json:"description" binding:"required"`
		PreparationTime  int                      `json:"preparation_time" binding:"required,min=1"`
		CookingTime      int                      `json:"cooking_time" binding:"required,min=0"`
		Servings         int                      `json:"servings" binding:"required,min=1"`
		DifficultyLevel  string                   `json:"difficulty_level" binding:"required,oneof=easy medium hard"`
		CategoryID       string                   `json:"category_id" binding:"required"`
		Price            float64                  `json:"price" binding:"min=0"`
		Ingredients      []models.Ingredient      `json:"ingredients" binding:"required,min=1"`
		Steps            []models.Step            `json:"steps" binding:"required,min=1"`
		FeaturedImageURL string                   `json:"featured_image_url"`
		Images           []models.RecipeImage     `json:"images"`
	}
	
	if err := c.ShouldBindJSON(&recipeInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Start transaction
	tx := h.DB.Begin()
	
	// Create recipe
	recipe := models.Recipe{
		Title:            recipeInput.Title,
		Description:      recipeInput.Description,
		PreparationTime:  recipeInput.PreparationTime,
		CookingTime:      recipeInput.CookingTime,
		Servings:         recipeInput.Servings,
		DifficultyLevel:  recipeInput.DifficultyLevel,
		CategoryID:       recipeInput.CategoryID,
		UserID:           userID.(string),
		Price:            recipeInput.Price,
		IsPublished:      true,
	}
	
	if err := tx.Create(&recipe).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create recipe"})
		return
	}
	
	// Create ingredients
	for i := range recipeInput.Ingredients {
		recipeInput.Ingredients[i].RecipeID = recipe.ID
		recipeInput.Ingredients[i].ID = "" // Ensure new ID is generated
	}
	
	if err := tx.Create(&recipeInput.Ingredients).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ingredients"})
		return
	}
	
	// Create steps
	for i := range recipeInput.Steps {
		recipeInput.Steps[i].RecipeID = recipe.ID
		recipeInput.Steps[i].ID = "" // Ensure new ID is generated
		recipeInput.Steps[i].StepNumber = i + 1
	}
	
	if err := tx.Create(&recipeInput.Steps).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create steps"})
		return
	}
	
	// Handle images
	if recipeInput.FeaturedImageURL != "" {
		featuredImage := models.RecipeImage{
			RecipeID:   recipe.ID,
			ImageURL:   recipeInput.FeaturedImageURL,
			IsFeatured: true,
		}
		if err := tx.Create(&featuredImage).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create featured image"})
			return
		}
		recipe.FeaturedImageURL = &recipeInput.FeaturedImageURL
		if err := tx.Save(&recipe).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update recipe with featured image"})
			return
		}
	}
	
	// Create additional images
	for i := range recipeInput.Images {
		recipeInput.Images[i].RecipeID = recipe.ID
		recipeInput.Images[i].ID = "" // Ensure new ID is generated
		if recipeInput.Images[i].ImageURL == recipeInput.FeaturedImageURL {
			recipeInput.Images[i].IsFeatured = true
		}
	}
	
	if len(recipeInput.Images) > 0 {
		if err := tx.Create(&recipeInput.Images).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create images"})
			return
		}
	}
	
	tx.Commit()
	
	// Load the complete recipe with relationships
	var createdRecipe models.Recipe
	if err := h.DB.Preload("User").Preload("Category").Preload("Ingredients").
		Preload("Steps").Preload("Images").First(&createdRecipe, "id = ?", recipe.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch created recipe"})
		return
	}
	
	c.JSON(http.StatusCreated, createdRecipe)
}

func (h *RecipeHandler) GetRecipes(c *gin.Context) {
	var filters models.SearchFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.Limit == 0 {
		filters.Limit = 12
	}
	
	offset := (filters.Page - 1) * filters.Limit
	
	query := h.DB.Preload("User").Preload("Category").Preload("Images").
		Where("is_published = ?", true)
	
	if filters.Query != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", 
			"%"+filters.Query+"%", "%"+filters.Query+"%")
	}
	
	if filters.CategoryID != "" {
		query = query.Where("category_id = ?", filters.CategoryID)
	}
	
	if filters.MaxTotalTime > 0 {
		query = query.Where("(preparation_time + cooking_time) <= ?", filters.MaxTotalTime)
	}
	
	if filters.MinRating > 0 {
		query = query.Where("average_rating >= ?", filters.MinRating)
	}
	
	if filters.Ingredient != "" {
		query = query.Joins("JOIN ingredients ON ingredients.recipe_id = recipes.id").
			Where("ingredients.name ILIKE ?", "%"+filters.Ingredient+"%")
	}
	
	var recipes []models.Recipe
	var total int64
	
	query.Model(&models.Recipe{}).Count(&total)
	
	if err := query.Offset(offset).Limit(filters.Limit).
		Order("created_at DESC").Find(&recipes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recipes"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"recipes": recipes,
		"total":   total,
		"page":    filters.Page,
		"limit":   filters.Limit,
		"pages":   (int(total) + filters.Limit - 1) / filters.Limit,
	})
}

func (h *RecipeHandler) GetRecipe(c *gin.Context) {
	recipeID := c.Param("id")
	
	var recipe models.Recipe
	if err := h.DB.Preload("User").Preload("Category").Preload("Ingredients").
		Preload("Steps", func(db *gorm.DB) *gorm.DB {
			return db.Order("steps.step_number ASC")
		}).Preload("Images").Preload("Comments", func(db *gorm.DB) *gorm.DB {
			return db.Preload("User").Order("comments.created_at DESC")
		}).First(&recipe, "id = ? AND is_published = ?", recipeID, true).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}
	
	// Check if user is authenticated and get their interactions
	userID, exists := c.Get("user_id")
	if exists {
		var userLike models.Like
		var userBookmark models.Bookmark
		var userRating models.Rating
		
		h.DB.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&userLike)
		h.DB.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&userBookmark)
		h.DB.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&userRating)
		
		recipeResponse := gin.H{
			"recipe":        recipe,
			"user_liked":    userLike.ID != "",
			"user_bookmarked": userBookmark.ID != "",
			"user_rating":   userRating.Rating,
		}
		
		c.JSON(http.StatusOK, recipeResponse)
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"recipe":        recipe,
		"user_liked":    false,
		"user_bookmarked": false,
		"user_rating":   0,
	})
}

func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	recipeID := c.Param("id")
	
	// Check if recipe exists and belongs to user
	var existingRecipe models.Recipe
	if err := h.DB.First(&existingRecipe, "id = ? AND user_id = ?", recipeID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found or access denied"})
		return
	}
	
	var updateData models.Recipe
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Update recipe
	if err := h.DB.Model(&existingRecipe).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update recipe"})
		return
	}
	
	c.JSON(http.StatusOK, existingRecipe)
}

func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	recipeID := c.Param("id")
	
	// Check if recipe exists and belongs to user
	var recipe models.Recipe
	if err := h.DB.First(&recipe, "id = ? AND user_id = ?", recipeID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found or access denied"})
		return
	}
	
	if err := h.DB.Delete(&recipe).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete recipe"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Recipe deleted successfully"})
}

func (h *RecipeHandler) ToggleLike(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	recipeID := c.Param("id")
	
	// Check if recipe exists
	var recipe models.Recipe
	if err := h.DB.First(&recipe, "id = ?", recipeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}
	
	var existingLike models.Like
	if err := h.DB.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&existingLike).Error; err != nil {
		// Like doesn't exist, create it
		like := models.Like{
			UserID:   userID.(string),
			RecipeID: recipeID,
		}
		
		if err := h.DB.Create(&like).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like recipe"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"liked": true, "message": "Recipe liked"})
		return
	}
	
	// Like exists, remove it
	if err := h.DB.Delete(&existingLike).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike recipe"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"liked": false, "message": "Recipe unliked"})
}

func (h *RecipeHandler) ToggleBookmark(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	recipeID := c.Param("id")
	
	// Check if recipe exists
	var recipe models.Recipe
	if err := h.DB.First(&recipe, "id = ?", recipeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}
	
	var existingBookmark models.Bookmark
	if err := h.DB.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&existingBookmark).Error; err != nil {
		// Bookmark doesn't exist, create it
		bookmark := models.Bookmark{
			UserID:   userID.(string),
			RecipeID: recipeID,
		}
		
		if err := h.DB.Create(&bookmark).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bookmark recipe"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"bookmarked": true, "message": "Recipe bookmarked"})
		return
	}
	
	// Bookmark exists, remove it
	if err := h.DB.Delete(&existingBookmark).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove bookmark"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"bookmarked": false, "message": "Bookmark removed"})
}

func (h *RecipeHandler) AddRating(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	recipeID := c.Param("id")
	
	var ratingInput struct {
		Rating int `json:"rating" binding:"required,min=1,max=5"`
	}
	
	if err := c.ShouldBindJSON(&ratingInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if recipe exists
	var recipe models.Recipe
	if err := h.DB.First(&recipe, "id = ?", recipeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}
	
	// Update or create rating
	var existingRating models.Rating
	if err := h.DB.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&existingRating).Error; err != nil {
		// Create new rating
		rating := models.Rating{
			UserID:   userID.(string),
			RecipeID: recipeID,
			Rating:   ratingInput.Rating,
		}
		
		if err := h.DB.Create(&rating).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add rating"})
			return
		}
	} else {
		// Update existing rating
		existingRating.Rating = ratingInput.Rating
		if err := h.DB.Save(&existingRating).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rating"})
			return
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Rating added successfully"})
}

func (h *RecipeHandler) AddComment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	recipeID := c.Param("id")
	
	var commentInput struct {
		Content string `json:"content" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&commentInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if recipe exists
	var recipe models.Recipe
	if err := h.DB.First(&recipe, "id = ?", recipeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}
	
	comment := models.Comment{
		UserID:   userID.(string),
		RecipeID: recipeID,
		Content:  commentInput.Content,
	}
	
	if err := h.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment"})
		return
	}
	
	// Load comment with user data
	h.DB.Preload("User").First(&comment, "id = ?", comment.ID)
	
	c.JSON(http.StatusCreated, comment)
}