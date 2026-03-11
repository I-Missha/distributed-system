package models

const (
	StatusInProgress = "IN_PROGRESS"
	StatusReady      = "READY"
	StatusError      = "ERROR"
)

type CrackRequest struct {
	Hash      string `json:"hash" binding:"required"`
	MaxLength int    `json:"maxLength" binding:"required,gt=0"`
}

type CrackResponse struct {
	RequestID string `json:"requestId"`
}

type StatusResponse struct {
	Status   string   `json:"status"`
	Progress int      `json:"progress"`
	Data     []string `json:"data"`
}

type WorkerTask struct {
	RequestID  string `json:"requestId"`
	Hash       string `json:"hash"`
	MaxLength  int    `json:"maxLength"`
	PartNumber int    `json:"partNumber"`
	TotalParts int    `json:"totalParts"`
	Alphabet   string `json:"alphabet"`
}

type WorkerResult struct {
	RequestID  string   `json:"requestId"`
	PartNumber int      `json:"partNumber"`
	FoundWords []string `json:"foundWords"`
	Error      bool     `json:"error"`
}
