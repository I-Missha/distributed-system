package manager

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"crackhash/internal/models"

	"github.com/google/uuid"
)

type Task struct {
	ID             string
	Status         string
	Progress       int
	Data           []string
	TotalParts     int
	PartsCompleted int
	Hash           string
	MaxLength      int
	mu             sync.Mutex
}

type Manager struct {
	tasks      map[string]*Task
	tasksMu    sync.RWMutex
	workerURLs []string
	alphabet   string
	httpClient *http.Client
}

func NewManager(workerURLs []string, alphabet string) *Manager {
	return &Manager{
		tasks:      make(map[string]*Task),
		workerURLs: workerURLs,
		alphabet:   alphabet,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

const magicConstant = 7

func (m *Manager) CreateTask(hash string, maxLength int) string {
	requestID := uuid.New().String()
	totalWorkers := len(m.workerURLs)

	task := &Task{
		ID:             requestID,
		Status:         models.StatusInProgress,
		Progress:       0,
		Data:           []string{},
		TotalParts:     totalWorkers * magicConstant,
		PartsCompleted: 0,
		Hash:           hash,
		MaxLength:      maxLength,
	}

	m.tasksMu.Lock()
	m.tasks[requestID] = task
	m.tasksMu.Unlock()

	log.Printf("Created task %s for hash %s", requestID, hash)

	go m.dispatchTasks(task)

	return requestID
}

func (m *Manager) GetStatus(requestID string) (models.StatusResponse, bool) {
	m.tasksMu.RLock()
	task, exists := m.tasks[requestID]
	m.tasksMu.RUnlock()

	if !exists {
		return models.StatusResponse{}, false
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	var data []string
	if task.Status == models.StatusReady {
		data = task.Data
	} else if task.Status == models.StatusInProgress {
		data = nil
	} else {
		data = task.Data
	}

	return models.StatusResponse{
		Status:   task.Status,
		Progress: task.Progress,
		Data:     data,
	}, true
}

func (m *Manager) UpdateTask(result models.WorkerResult) {
	m.tasksMu.RLock()
	task, exists := m.tasks[result.RequestID]
	m.tasksMu.RUnlock()

	if !exists {
		log.Printf("Warning: Received result for unknown task %s", result.RequestID)
		return
	}

	task.mu.Lock()
	defer task.mu.Unlock()

	if task.Status == models.StatusError || task.Status == models.StatusReady {
		return
	}

	if result.Error {
		task.Status = models.StatusError
		log.Printf("Task %s encountered an error from part %d", task.ID, result.PartNumber)
		return
	}

	if len(result.FoundWords) > 0 {
		task.Data = append(task.Data, result.FoundWords...)
	}

	task.PartsCompleted++
	task.Progress = (task.PartsCompleted * 100) / task.TotalParts

	log.Printf("Task %s: Part %d completed. Progress: %d%%", task.ID, result.PartNumber, task.Progress)

	if task.PartsCompleted == task.TotalParts {
		task.Status = models.StatusReady
		task.Progress = 100
		log.Printf("Task %s finished. Found %d words.", task.ID, len(task.Data))
	}
}

func (m *Manager) dispatchTasks(task *Task) {
	sentTasks := 0
	for sentTasks < task.TotalParts {
		for _, workerURL := range m.workerURLs {
			if sentTasks >= task.TotalParts {
				break
			}
			workerTask := models.WorkerTask{
				RequestID:  task.ID,
				Hash:       task.Hash,
				MaxLength:  task.MaxLength,
				PartNumber: sentTasks,
				TotalParts: task.TotalParts,
				Alphabet:   m.alphabet,
			}

			go func(url string, wt models.WorkerTask) {
				err := m.sendToWorker(url, wt)
				if err != nil {
					log.Printf("Failed to dispatch task %s part %d to %s: %v", wt.RequestID, wt.PartNumber, url, err)

					m.UpdateTask(models.WorkerResult{
						RequestID:  wt.RequestID,
						PartNumber: wt.PartNumber,
						Error:      true,
					})
				}
			}(workerURL, workerTask)
			sentTasks++
		}
	}

}

func (m *Manager) sendToWorker(workerURL string, task models.WorkerTask) error {
	jsonData, err := json.Marshal(task)
	if err != nil {
		return err
	}

	url := workerURL + "/internal/api/worker/hash/crack/task"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		log.Printf("Worker %s returned status: %d", url, resp.StatusCode)
	}

	return nil
}
