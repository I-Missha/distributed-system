package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"crackhash/internal/generator"
	"crackhash/internal/models"

	_ "crackhash/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Worker Internal API
// @version         1.0
// @description     This is an internal API for the CrackHash Worker.

func main() {
	port := os.Getenv("WORKER_PORT")
	if port == "" {
		port = "8081"
	}

	router := gin.Default()

	router.POST("/internal/api/worker/hash/crack/task", handleTask)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Worker starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}
}

// @Summary Receive a hash cracking task from the manager
// @Description Receives a task with hash, max length, alphabet, and partition info, then starts processing asynchronously.
// @Accept json
// @Produce json
// @Param task body models.WorkerTask true "Task Data"
// @Success 200 "Task accepted"
// @Failure 400 "Invalid request body"
// @Router /internal/api/worker/hash/crack/task [post]
func handleTask(c *gin.Context) {
	var task models.WorkerTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	managerURL := os.Getenv("MANAGER_URL")
	if managerURL == "" {
		managerURL = "http://manager:8080"
	}

	go processTask(task, managerURL)

	c.Status(http.StatusOK)
}

func processTask(task models.WorkerTask, managerURL string) {
	log.Printf("Starting task %s (part %d/%d)", task.RequestID, task.PartNumber, task.TotalParts)

	foundWords := generator.GenerateAndMatch(task.Hash, task.MaxLength, task.Alphabet, task.PartNumber, task.TotalParts)

	log.Printf("Finished task %s (part %d). Found %d words.", task.RequestID, task.PartNumber, len(foundWords))

	result := models.WorkerResult{
		RequestID:  task.RequestID,
		PartNumber: task.PartNumber,
		FoundWords: foundWords,
		Error:      false,
	}

	sendResultToManager(result, managerURL)
}

func sendResultToManager(result models.WorkerResult, managerURL string) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal result: %v", err)
		return
	}

	url := managerURL + "/internal/api/manager/hash/crack/request"
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create request to manager: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send result to manager: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Manager returned non-OK status: %d", resp.StatusCode)
	}
}
