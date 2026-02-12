package cluster

import (
	"encoding/json"
	"testing"
)

func TestNewEnvelope(t *testing.T) {
	req := ApplyRequest{
		Agents: []AgentDef{
			{Name: "watcher", Definition: "(defagent \"watcher\")", ID: "abc"},
		},
	}

	env, err := NewEnvelope(MsgApplyRequest, req)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	if env.Type != MsgApplyRequest {
		t.Fatalf("expected type %s, got %s", MsgApplyRequest, env.Type)
	}

	// Verify round-trip.
	var decoded ApplyRequest
	if err := env.DecodePayload(&decoded); err != nil {
		t.Fatalf("DecodePayload: %v", err)
	}
	if len(decoded.Agents) != 1 || decoded.Agents[0].Name != "watcher" {
		t.Fatalf("unexpected decoded agents: %+v", decoded.Agents)
	}
}

func TestEnvelopeJSON(t *testing.T) {
	req := ApplyRequest{
		Agents: []AgentDef{
			{Name: "test", Definition: "(defagent \"test\")", ID: "xyz"},
		},
	}

	env, err := NewEnvelope(MsgApplyRequest, req)
	if err != nil {
		t.Fatal(err)
	}

	// Marshal to JSON and back.
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var env2 Envelope
	if err := json.Unmarshal(data, &env2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env2.Type != MsgApplyRequest {
		t.Fatalf("expected type %s after round-trip, got %s", MsgApplyRequest, env2.Type)
	}

	var decoded ApplyRequest
	if err := env2.DecodePayload(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Agents[0].ID != "xyz" {
		t.Fatalf("expected ID xyz, got %s", decoded.Agents[0].ID)
	}
}

func TestApplyResponseEnvelope(t *testing.T) {
	resp := ApplyResponse{
		Summary: ApplySummary{
			Created:   []string{"alpha"},
			Updated:   []string{"beta"},
			Unchanged: []string{"gamma"},
		},
	}

	env, err := NewEnvelope(MsgApplyResponse, resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded ApplyResponse
	if err := env.DecodePayload(&decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Summary.Created) != 1 || decoded.Summary.Created[0] != "alpha" {
		t.Fatalf("unexpected created: %v", decoded.Summary.Created)
	}
}

func TestSteerInjectEnvelope(t *testing.T) {
	req := SteerInjectRequest{
		AgentName: "watcher",
		StepLabel: "observe",
		Iteration: 3,
		Message:   "focus on errors",
	}

	env, err := NewEnvelope(MsgSteerInject, req)
	if err != nil {
		t.Fatal(err)
	}

	var decoded SteerInjectRequest
	if err := env.DecodePayload(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.AgentName != "watcher" || decoded.Iteration != 3 {
		t.Fatalf("unexpected: %+v", decoded)
	}
}
