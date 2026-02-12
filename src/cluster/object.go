// Package cluster implements the gcluster control plane: cluster objects,
// state management, persistence, network protocol, and agent execution.
//
// A ClusterObject represents a deployed agent definition. Each object has a
// stable identity derived from the SHA-256 of its canonical S-expression.
// When an agent's definition changes, a new Revision is appended; existing
// runs continue on their current revision until stopped.
package cluster

import "time"

// RunState represents the lifecycle state of a cluster object.
type RunState string

const (
	RunStatePending RunState = "pending"
	RunStateRunning RunState = "running"
	RunStateStopped RunState = "stopped"
)

// Revision captures a point-in-time snapshot of an agent definition.
type Revision struct {
	// ID is the full SHA-256 hex of the canonical S-expression for this revision.
	ID string `json:"id"`
	// Timestamp is when this revision was created.
	Timestamp time.Time `json:"timestamp"`
	// Definition is the canonical S-expression string.
	Definition string `json:"definition"`
}

// ClusterObject is the fundamental unit of cluster state. It tracks a named
// agent across revisions and run states. The cluster is additive-only: objects
// are never deleted, only updated with new revisions.
type ClusterObject struct {
	// ID is the stable SHA-256 hex of the *current* canonical S-expression.
	ID string `json:"id"`
	// Name is the agent name (suffix after "agent-" in the .p source).
	Name string `json:"name"`
	// Definition is the current canonical S-expression string.
	Definition string `json:"definition"`
	// Revisions is an ordered list of all revisions, oldest first.
	Revisions []Revision `json:"revisions"`
	// State is the current run state (pending, running, stopped).
	State RunState `json:"state"`
	// CurrentRevision points to the active revision's ID.
	CurrentRevision string `json:"current_revision"`
}

// AgentDef is the payload for an agent definition sent from apply to master.
// It contains all the information needed to create or update a ClusterObject.
type AgentDef struct {
	// Name is the agent name (without "agent-" prefix).
	Name string `json:"name"`
	// Definition is the canonical S-expression string.
	Definition string `json:"definition"`
	// ID is the full SHA-256 hex of the definition.
	ID string `json:"id"`
}

// ApplySummary reports the outcome of an apply operation.
type ApplySummary struct {
	Created   []string `json:"created"`   // Names of newly created agents.
	Updated   []string `json:"updated"`   // Names of agents with new revisions.
	Unchanged []string `json:"unchanged"` // Names of agents whose definitions didn't change.
}
