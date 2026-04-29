package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CrackRequest struct {
	Hash      string `json:"Hash"`
	MaxLength int    `json:"MaxLength"`
}

type CrackResponse struct {
	RequestID string `json:"RequestID"`
}

type StatusResponse struct {
	Status   string   `json:"Status"`
	Progress int      `json:"Progress"`
	Data     []string `json:"Data"`
}

var taskStore = make(map[string]string)

func main() {
	port := os.Getenv("FRONTEND_PORT")
	if port == "" {
		port = "80"
	}

	managerURL := os.Getenv("MANAGER_URL")
	if managerURL == "" {
		managerURL = "http://manager:8080"
	}

	router := gin.Default()
	router.LoadHTMLGlob("templates/*")

	// Render the main page
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Handle form submission to create a task
	router.POST("/crack", func(c *gin.Context) {
		hash := c.PostForm("hash")
		maxLengthStr := c.PostForm("maxLength")

		maxLength, err := strconv.Atoi(maxLengthStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Invalid max length")
			return
		}

		reqBody := CrackRequest{
			Hash:      hash,
			MaxLength: maxLength,
		}

		jsonData, _ := json.Marshal(reqBody)
		resp, err := http.Post(managerURL+"/api/hash/crack", "application/json", bytes.NewBuffer(jsonData))
		if err != nil || resp.StatusCode != http.StatusOK {
			log.Printf("Failed to create task: %v", err)
			c.String(http.StatusInternalServerError, "Failed to communicate with Manager")
			return
		}
		defer resp.Body.Close()

		var crackResp CrackResponse
		bodyBytes, _ := io.ReadAll(resp.Body)
		json.Unmarshal(bodyBytes, &crackResp)

		taskStore[crackResp.RequestID] = hash
		shortID := crackResp.RequestID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		// Return a task card component
		c.HTML(http.StatusOK, "task.html", gin.H{
			"RequestID": crackResp.RequestID,
			"ShortID":   shortID,
			"Hash":      hash,
			"Status":    "IN_PROGRESS",
			"Progress":  0,
		})
	})

	// Handle status polling
	router.GET("/status", func(c *gin.Context) {
		requestID := c.Query("requestId")
		if requestID == "" {
			c.String(http.StatusBadRequest, "Missing requestId")
			return
		}

		resp, err := http.Get(fmt.Sprintf("%s/api/hash/status?requestId=%s", managerURL, requestID))
		if err != nil {
			log.Printf("Failed to get status: %v", err)
			c.String(http.StatusInternalServerError, "Failed to get status from Manager")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			c.String(http.StatusNotFound, "Task not found")
			return
		}

		var statusResp StatusResponse
		bodyBytes, _ := io.ReadAll(resp.Body)
		json.Unmarshal(bodyBytes, &statusResp)

		hash := taskStore[requestID]
		shortID := requestID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}

		c.HTML(http.StatusOK, "task.html", gin.H{
			"RequestID": requestID,
			"ShortID":   shortID,
			"Hash":      hash,
			"Status":    statusResp.Status,
			"Progress":  statusResp.Progress,
			"Data":      statusResp.Data,
		})
	})

	log.Printf("Frontend starting on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start frontend: %v", err)
	}
}
