package models

import (
	"time"
	
	"gorm.io/gorm"
)

type User struct {
	ID           string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	Username     string    `json:"username" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	AvatarURL    *string   `json:"avatar_url"`
	Bio          *string   `json:"bio"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Recipes      []Recipe  `json:"recipes" gorm:"foreignKey:UserID"`
}

type Category struct {
	ID          string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Description *string   `json:"description"`
	ImageURL    *string   `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
	Recipes     []Recipe  `json:"recipes" gorm:"foreignKey:CategoryID"`
}

type Recipe struct {
	ID               string         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Title            string         `json:"title" gorm:"not null"`
	Description      string         `json:"description"`
	FeaturedImageURL *string        `json:"featured_image_url"`
	PreparationTime  int            `json:"preparation_time" gorm:"not null"`
	CookingTime      int            `json:"cooking_time" gorm:"not null"`
	Servings         int            `json:"servings" gorm:"not null"`
	DifficultyLevel  string         `json:"difficulty_level" gorm:"type:varchar(20)"`
	CategoryID       string         `json:"category_id" gorm:"type:uuid;not null"`
	UserID           string         `json:"user_id" gorm:"type:uuid;not null"`
	Price            float64        `json:"price" gorm:"type:decimal(10,2);default:0"`
	AverageRating    float64        `json:"average_rating" gorm:"type:decimal(3,2);default:0"`
	TotalRatings     int            `json:"total_ratings" gorm:"default:0"`
	LikeCount        int            `json:"like_count" gorm:"default:0"`
	IsPublished      bool           `json:"is_published" gorm:"default:false"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	
	// Relationships
	User         User            `json:"user" gorm:"foreignKey:UserID"`
	Category     Category        `json:"category" gorm:"foreignKey:CategoryID"`
	Ingredients  []Ingredient    `json:"ingredients" gorm:"foreignKey:RecipeID"`
	Steps        []Step          `json:"steps" gorm:"foreignKey:RecipeID"`
	Images       []RecipeImage   `json:"images" gorm:"foreignKey:RecipeID"`
	Likes        []Like          `json:"likes" gorm:"foreignKey:RecipeID"`
	Bookmarks    []Bookmark      `json:"bookmarks" gorm:"foreignKey:RecipeID"`
	Comments     []Comment       `json:"comments" gorm:"foreignKey:RecipeID"`
	Ratings      []Rating        `json:"ratings" gorm:"foreignKey:RecipeID"`
}

type Ingredient struct {
	ID        string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	RecipeID  string    `json:"recipe_id" gorm:"type:uuid;not null"`
	Name      string    `json:"name" gorm:"not null"`
	Quantity  string    `json:"quantity"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"created_at"`
}

type Step struct {
	ID          string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	RecipeID    string    `json:"recipe_id" gorm:"type:uuid;not null"`
	StepNumber  int       `json:"step_number" gorm:"not null"`
	Instruction string    `json:"instruction" gorm:"not null"`
	ImageURL    *string   `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type RecipeImage struct {
	ID           string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	RecipeID     string    `json:"recipe_id" gorm:"type:uuid;not null"`
	ImageURL     string    `json:"image_url" gorm:"not null"`
	IsFeatured   bool      `json:"is_featured" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
}

type Like struct {
	ID        string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID    string    `json:"user_id" gorm:"type:uuid;not null"`
	RecipeID  string    `json:"recipe_id" gorm:"type:uuid;not null"`
	CreatedAt time.Time `json:"created_at"`
	
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Recipe Recipe `json:"recipe" gorm:"foreignKey:RecipeID"`
}

type Bookmark struct {
	ID        string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID    string    `json:"user_id" gorm:"type:uuid;not null"`
	RecipeID  string    `json:"recipe_id" gorm:"type:uuid;not null"`
	CreatedAt time.Time `json:"created_at"`
	
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Recipe Recipe `json:"recipe" gorm:"foreignKey:RecipeID"`
}

type Comment struct {
	ID        string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID    string    `json:"user_id" gorm:"type:uuid;not null"`
	RecipeID  string    `json:"recipe_id" gorm:"type:uuid;not null"`
	Content   string    `json:"content" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Recipe Recipe `json:"recipe" gorm:"foreignKey:RecipeID"`
}

type Rating struct {
	ID        string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID    string    `json:"user_id" gorm:"type:uuid;not null"`
	RecipeID  string    `json:"recipe_id" gorm:"type:uuid;not null"`
	Rating    int       `json:"rating" gorm:"not null;check:rating>=1 AND rating<=5"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Recipe Recipe `json:"recipe" gorm:"foreignKey:RecipeID"`
}

type Purchase struct {
	ID                  string    `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID              string    `json:"user_id" gorm:"type:uuid;not null"`
	RecipeID            string    `json:"recipe_id" gorm:"type:uuid;not null"`
	Amount              float64   `json:"amount" gorm:"type:decimal(10,2);not null"`
	ChapaTransactionID  *string   `json:"chapa_transaction_id"`
	Status              string    `json:"status" gorm:"default:'pending'"`
	CreatedAt           time.Time `json:"created_at"`
	
	User   User   `json:"user" gorm:"foreignKey:UserID"`
	Recipe Recipe `json:"recipe" gorm:"foreignKey:RecipeID"`
}

// Auth types
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// Search types
type SearchFilters struct {
	Query         string  `form:"q"`
	CategoryID    string  `form:"category_id"`
	MaxTotalTime  int     `form:"max_total_time"`
	Ingredient    string  `form:"ingredient"`
	MinRating     float64 `form:"min_rating"`
	Page          int     `form:"page" binding:"min=1"`
	Limit         int     `form:"limit" binding:"min=1,max=50"`
}