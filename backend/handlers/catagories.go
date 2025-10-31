package handlers

import (
	"net/http"
	
	"food-recipes-backend/models"
	
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CategoryHandler struct {
	DB *gorm.DB
}

func NewCategoryHandler(db *gorm.DB) *CategoryHandler {
	return &CategoryHandler{DB: db}
}

func (h *CategoryHandler) GetCategories(c *gin.Context) {
	var categories []models.Category
	
	if err := h.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	
	c.JSON(http.StatusOK, categories)
}

func (h *CategoryHandler) GetCategoryRecipes(c *gin.Context) {
	categoryID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 12
	}
	
	offset := (page - 1) * limit
	
	var category models.Category
	if err := h.DB.First(&category, "id = ?", categoryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}
	
	var recipes []models.Recipe
	var total int64
	
	h.DB.Model(&models.Recipe{}).Where("category_id = ? AND is_published = ?", categoryID, true).Count(&total)
	
	if err := h.DB.Preload("User").Preload("Category").Preload("Images").
		Where("category_id = ? AND is_published = ?", categoryID, true).
		Offset(offset).Limit(limit).
		Order("created_at DESC").Find(&recipes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recipes"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"recipes":  recipes,
		"total":    total,
		"page":     page,
		"limit":    limit,
		"pages":    (int(total) + limit - 1) / limit,
	})
}