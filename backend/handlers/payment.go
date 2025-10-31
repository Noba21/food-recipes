package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	
	"food-recipes-backend/models"
	
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChapaPaymentHandler struct {
	DB          *gorm.DB
	ChapaSecret string
}

func NewChapaPaymentHandler(db *gorm.DB, chapaSecret string) *ChapaPaymentHandler {
	return &ChapaPaymentHandler{
		DB:          db,
		ChapaSecret: chapaSecret,
	}
}

type ChapaInitializeRequest struct {
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	Email          string `json:"email"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Phone          string `json:"phone,omitempty"`
	TxRef          string `json:"tx_ref"`
	CallbackURL    string `json:"callback_url"`
	ReturnURL      string `json:"return_url"`
	CustomTitle    string `json:"custom_title,omitempty"`
	CustomDescription string `json:"custom_description,omitempty"`
}

type ChapaInitializeResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Data    struct {
		CheckoutURL string `json:"checkout_url"`
	} `json:"data"`
}

type ChapaVerifyResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Data    struct {
		Status   string `json:"status"`
		TxRef    string `json:"tx_ref"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	} `json:"data"`
}

func (h *ChapaPaymentHandler) InitializePayment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	var paymentRequest struct {
		RecipeID string  `json:"recipe_id" binding:"required"`
		Amount   float64 `json:"amount" binding:"required,min=0.01"`
	}
	
	if err := c.ShouldBindJSON(&paymentRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Check if recipe exists and get details
	var recipe models.Recipe
	if err := h.DB.First(&recipe, "id = ?", paymentRequest.RecipeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Recipe not found"})
		return
	}
	
	// Check if user already purchased this recipe
	var existingPurchase models.Purchase
	if err := h.DB.Where("user_id = ? AND recipe_id = ?", userID, paymentRequest.RecipeID).First(&existingPurchase).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already purchased this recipe"})
		return
	}
	
	// Get user details
	var user models.User
	if err := h.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	
	// Generate unique transaction reference
	txRef := fmt.Sprintf("recipe-%s-%d", paymentRequest.RecipeID, time.Now().UnixNano())
	
	// Create purchase record
	purchase := models.Purchase{
		UserID:     userID.(string),
		RecipeID:   paymentRequest.RecipeID,
		Amount:     paymentRequest.Amount,
		Status:     "pending",
	}
	
	if err := h.DB.Create(&purchase).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create purchase record"})
		return
	}
	
	// Initialize Chapa payment
	chapaRequest := ChapaInitializeRequest{
		Amount:      fmt.Sprintf("%.2f", paymentRequest.Amount),
		Currency:    "ETB",
		Email:       user.Email,
		FirstName:   user.Username,
		LastName:    "User",
		TxRef:       txRef,
		CallbackURL: "http://localhost:8080/api/payment/verify",
		ReturnURL:   "http://localhost:3000/payment/success",
		CustomTitle: "Food Recipe Purchase",
		CustomDescription: fmt.Sprintf("Purchase of recipe: %s", recipe.Title),
	}
	
	jsonData, err := json.Marshal(chapaRequest)
	if err != nil {
		h.DB.Delete(&purchase) // Clean up failed purchase record
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare payment"})
		return
	}
	
	req, err := http.NewRequest("POST", "https://api.chapa.co/v1/transaction/initialize", bytes.NewBuffer(jsonData))
	if err != nil {
		h.DB.Delete(&purchase)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize payment"})
		return
	}
	
	req.Header.Set("Authorization", "Bearer "+h.ChapaSecret)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		h.DB.Delete(&purchase)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.DB.Delete(&purchase)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read payment response"})
		return
	}
	
	var chapaResponse ChapaInitializeResponse
	if err := json.Unmarshal(body, &chapaResponse); err != nil {
		h.DB.Delete(&purchase)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse payment response"})
		return
	}
	
	if chapaResponse.Status != "success" {
		h.DB.Delete(&purchase)
		c.JSON(http.StatusBadRequest, gin.H{"error": chapaResponse.Message})
		return
	}
	
	// Update purchase with transaction reference
	purchase.ChapaTransactionID = &txRef
	h.DB.Save(&purchase)
	
	c.JSON(http.StatusOK, gin.H{
		"checkout_url": chapaResponse.Data.CheckoutURL,
		"purchase_id":  purchase.ID,
	})
}

func (h *ChapaPaymentHandler) VerifyPayment(c *gin.Context) {
	txRef := c.Query("tx_ref")
	
	if txRef == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction reference required"})
		return
	}
	
	// Verify payment with Chapa
	req, err := http.NewRequest("GET", "https://api.chapa.co/v1/transaction/verify/"+txRef, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}
	
	req.Header.Set("Authorization", "Bearer "+h.ChapaSecret)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment verification service unavailable"})
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read verification response"})
		return
	}
	
	var verifyResponse ChapaVerifyResponse
	if err := json.Unmarshal(body, &verifyResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse verification response"})
		return
	}
	
	// Find and update purchase record
	var purchase models.Purchase
	if err := h.DB.Where("chapa_transaction_id = ?", txRef).First(&purchase).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Purchase record not found"})
		return
	}
	
	if verifyResponse.Data.Status == "success" {
		purchase.Status = "completed"
	} else {
		purchase.Status = "failed"
	}
	
	h.DB.Save(&purchase)
	
	c.JSON(http.StatusOK, gin.H{
		"status":  purchase.Status,
		"message": "Payment verification completed",
	})
}

func (h *ChapaPaymentHandler) GetUserPurchases(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	
	var purchases []models.Purchase
	if err := h.DB.Preload("Recipe").Preload("Recipe.User").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&purchases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch purchases"})
		return
	}
	
	c.JSON(http.StatusOK, purchases)
}