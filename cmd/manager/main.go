package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"crackhash/internal/manager"
	"crackhash/internal/models"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("MANAGER_PORT")
	if port == "" {
		port = "8080"
	}

	workerURLsStr := os.Getenv("WORKER_URLS")
	if workerURLsStr == "" {
		workerURLsStr = "http://worker1:8081,http://worker2:8081,http://worker3:8081"
	}
	workerURLs := strings.Split(workerURLsStr, ",")

	alphabet := os.Getenv("ALPHABET")
	if alphabet == "" {
		alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	}

	mgr := manager.NewManager(workerURLs, alphabet)
	router := gin.Default()

	router.POST("/api/hash/crack", func(c *gin.Context) {
		var req models.CrackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		requestID := mgr.CreateTask(req.Hash, req.MaxLength)
		c.JSON(http.StatusOK, models.CrackResponse{RequestID: requestID})
	})

	router.GET("/api/hash/status", func(c *gin.Context) {
		requestID := c.Query("requestId")
		if requestID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "requestId is required"})
			return
		}

		status, exists := mgr.GetStatus(requestID)
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
			return
		}

		c.JSON(http.StatusOK, status)
	})

	router.PATCH("/internal/api/manager/hash/crack/request", func(c *gin.Context) {
		var result models.WorkerResult
		if err := c.ShouldBindJSON(&result); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		mgr.UpdateTask(result)
		c.Status(http.StatusOK)
	})

	log.Printf("Manager starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start manager: %v", err)
	}
}
