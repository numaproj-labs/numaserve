package common

import (
	"encoding/json"
	"fmt"
)

type Metadata struct {
	Sender    string `json:"sender"`
	RequestID string `json:"requestID"`
}
type MLServeRequest struct {
	Metadata Metadata `json:"metadata"`
	Payload  []byte   `json:"payload"`
}
type MLServeResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Payload []byte `json:"payload"`
}

func (req *MLServeRequest) ToJSON() string {
	content, err := json.Marshal(req)
	if err != nil {
		return string("")
	}
	return string(content)
}

func (req *MLServeRequest) GetDestination() string {
	return fmt.Sprintf("%s/%s", req.Metadata.Sender, req.Metadata.RequestID)
}
