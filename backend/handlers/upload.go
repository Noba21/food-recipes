package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
	
	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	UploadDir string
}

func NewUploadHandler(uploadDir string) *UploadHandler {
	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create upload directory: %v", err))
	}
	
	return &UploadHandler{UploadDir: uploadDir}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}
	defer file.Close()
	
	// Validate file type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
		return
	}
	
	fileType := http.DetectContentType(buffer)
	if fileType != "image/jpeg" && fileType != "image/png" && fileType != "image/gif" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPEG, PNG, and GIF images are allowed"})
		return
	}
	
	// Reset file pointer
	_, err = file.Seek(0, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
		return
	}
	
	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		// Determine extension from content type
		switch fileType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		}
	}
	
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filepath := filepath.Join(h.UploadDir, filename)
	
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer out.Close()
	
	// Copy the file content
	_, err = io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	
	// Return the file URL (you might want to use a CDN URL in production)
	fileURL := fmt.Sprintf("/uploads/%s", filename)
	
	c.JSON(http.StatusOK, gin.H{
		"url":       fileURL,
		"filename":  filename,
		"file_size": header.Size,
		"mime_type": fileType,
	})
}

func (h *UploadHandler) ServeUploads(c *gin.Context) {
	filename := c.Param("filename")
	filepath := filepath.Join(h.UploadDir, filename)
	
	// Security check to prevent directory traversal
	if filepath != filepath.Clean(filepath) || filepath != filepath.Join(h.UploadDir, filename) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}
	
	c.File(filepath)
}