package cluster

import "encoding/json"

// DefaultAddr is the address the master listens on and clients connect to.
const DefaultAddr = "127.0.0.1:43252"

// MessageType identifies the kind of protocol message.
type MessageType string

const (
	MsgApplyRequest    MessageType = "apply_request"
	MsgApplyResponse   MessageType = "apply_response"
	MsgSteerSubscribe  MessageType = "steer_subscribe"
	MsgSteerState      MessageType = "steer_state"
	MsgSteerInject     MessageType = "steer_inject"
	MsgShutdownNotice  MessageType = "shutdown_notice"
)

// Envelope wraps every protocol message. Clients and server exchange
// newline-delimited JSON objects; each must include a "type" field.
type Envelope struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// --- Request / Response payloads ---

// ApplyRequest is sent by `gcluster apply` to submit agent definitions.
type ApplyRequest struct {
	Agents []AgentDef `json:"agents"`
}

// ApplyResponse is the master's reply to an apply request.
type ApplyResponse struct {
	Summary ApplySummary `json:"summary"`
	Error   string       `json:"error,omitempty"`
}

// SteerSubscribeRequest is sent by `gcluster steer` to begin receiving state.
type SteerSubscribeRequest struct{}

// SteerStatePayload pushes full cluster state to a steer client.
// Objects contains the declarative state (definitions, revisions, run state).
// Runs contains runtime iteration data for running agents — this is ephemeral
// and NOT persisted to disk, only sent to steer clients for observation.
// Methods contains resolved method bodies per agent (agent name → method name → body).
// Pipelines contains pipeline structure per agent (agent name → PipelineDef).
// Both are populated from the server's cache (set at apply time) so the TUI
// can display human-readable method text and pipeline-aware tree structure.
type SteerStatePayload struct {
	Objects   []ClusterObject                `json:"objects"`
	Runs      map[string]AgentRunSnapshot    `json:"runs,omitempty"`
	Methods   map[string]map[string]string   `json:"methods,omitempty"`
	Pipelines map[string]*PipelineDef        `json:"pipelines,omitempty"`
}

// SteerInjectRequest sends a human message into an agent's conversation.
type SteerInjectRequest struct {
	AgentName string `json:"agent_name"`
	StepLabel string `json:"step_label"`
	Iteration int    `json:"iteration"`
	Message   string `json:"message"`
}

// ShutdownNoticePayload notifies clients the master is shutting down.
type ShutdownNoticePayload struct {
	Reason string `json:"reason"`
}

// --- Helpers for encoding/decoding ---

// NewEnvelope creates an Envelope with the given type and marshalled payload.
func NewEnvelope(msgType MessageType, payload interface{}) (*Envelope, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		Type:    msgType,
		Payload: json.RawMessage(data),
	}, nil
}

// DecodePayload unmarshals the Envelope's payload into dst.
func (e *Envelope) DecodePayload(dst interface{}) error {
	return json.Unmarshal(e.Payload, dst)
}
