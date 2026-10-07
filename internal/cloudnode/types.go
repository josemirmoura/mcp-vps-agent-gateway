package cloudnode

import (
	"encoding/json"
	"time"
)

const ProtocolVersion = "1"

type Identity struct {
	NodeID      string `json:"node_id"`
	WorkspaceID string `json:"workspace_id"`
	PrivateKey  string `json:"private_key"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}


type PendingEnrollment struct {
	Token    string `json:"token"`
	NodeName string `json:"node_name"`
	Platform string `json:"platform"`
}

type NodeState struct {
	Identity   Identity           `json:"identity"`
	Enrollment *PendingEnrollment `json:"enrollment,omitempty"`
}

type EnrollmentRequest struct {
	Token             string `json:"token"`
	Name              string `json:"name"`
	Platform          string `json:"platform"`
	PublicKey         string `json:"public_key"`
	ProtocolVersion   string `json:"protocol_version,omitempty"`
	CapabilityVersion string `json:"capability_version,omitempty"`
}

type EnrollmentResponse struct {
	NodeID      string `json:"node_id"`
	WorkspaceID string `json:"workspace_id"`
	Fingerprint string `json:"fingerprint"`
}

type Task struct {
	ID                string          `json:"id"`
	WorkspaceID       string          `json:"workspace_id"`
	DestinationNodeID string          `json:"destination_node_id"`
	AgentID           string          `json:"agent_id,omitempty"`
	RequestedBy       string          `json:"requested_by"`
	Operation         string          `json:"operation"`
	Resource          string          `json:"resource,omitempty"`
	Action            string          `json:"action,omitempty"`
	GrantID           string          `json:"grant_id,omitempty"`
	Input             json.RawMessage `json:"input"`
	CapabilityScope   json.RawMessage `json:"capability_scope"`
	State             string          `json:"state"`
	LeaseID           string          `json:"lease_id"`
	LeaseExpiresAt    *time.Time      `json:"lease_expires_at,omitempty"`
	ExpiresAt         *time.Time      `json:"expires_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

type Completion struct {
	LeaseID string          `json:"lease_id"`
	Result  json.RawMessage `json:"result"`
	Error   string          `json:"error,omitempty"`
}
