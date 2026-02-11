package pipeline

import (
	"testing"
)

func TestParseBareSteps(t *testing.T) {
	p, err := Parse("idea -> spec -> plan -> loop(build)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if p.InitialInput != "idea" {
		t.Errorf("InitialInput = %q, want %q", p.InitialInput, "idea")
	}
	if len(p.Steps) != 3 {
		t.Fatalf("got %d steps, want 3", len(p.Steps))
	}

	// spec — bare word
	if p.Steps[0].Label != "spec" || p.Steps[0].Method != "spec" || p.Steps[0].Kind != StepSimple {
		t.Errorf("step 0: got %+v", p.Steps[0])
	}
	// plan — bare word
	if p.Steps[1].Label != "plan" || p.Steps[1].Method != "plan" || p.Steps[1].Kind != StepSimple {
		t.Errorf("step 1: got %+v", p.Steps[1])
	}
	// loop(build)
	if p.Steps[2].Kind != StepLoop || p.Steps[2].LoopMethod != "build" {
		t.Errorf("step 2: got %+v", p.Steps[2])
	}
}

func TestParseBookPipeline(t *testing.T) {
	p, err := Parse("topic -> brief (book-idea) -> chapter-outline (generate-chapter-index) -> chapters (map(chapters, flesh-out-chapter)) -> final (concat)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if p.InitialInput != "topic" {
		t.Errorf("InitialInput = %q, want %q", p.InitialInput, "topic")
	}
	if len(p.Steps) != 4 {
		t.Fatalf("got %d steps, want 4", len(p.Steps))
	}

	if p.Steps[0].Label != "brief" || p.Steps[0].Method != "book-idea" || p.Steps[0].Kind != StepSimple {
		t.Errorf("step 0: got %+v", p.Steps[0])
	}
	if p.Steps[1].Label != "chapter-outline" || p.Steps[1].Method != "generate-chapter-index" || p.Steps[1].Kind != StepSimple {
		t.Errorf("step 1: got %+v", p.Steps[1])
	}
	if p.Steps[2].Kind != StepMap || p.Steps[2].Label != "chapters" || p.Steps[2].MapRef != "chapters" || p.Steps[2].MapMethod != "flesh-out-chapter" {
		t.Errorf("step 2: got %+v", p.Steps[2])
	}
	if p.Steps[3].Label != "final" || p.Steps[3].Method != "concat" || p.Steps[3].Kind != StepSimple {
		t.Errorf("step 3: got %+v", p.Steps[3])
	}
}

func TestParseLabeledSteps(t *testing.T) {
	p, err := Parse("topic -> outlined (detail) -> chapters (map(chapters, flesh-out))")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if p.InitialInput != "topic" {
		t.Errorf("InitialInput = %q, want %q", p.InitialInput, "topic")
	}
	if len(p.Steps) != 2 {
		t.Fatalf("got %d steps, want 2", len(p.Steps))
	}

	if p.Steps[0].Label != "outlined" || p.Steps[0].Method != "detail" || p.Steps[0].Kind != StepSimple {
		t.Errorf("step 0: got %+v", p.Steps[0])
	}
	if p.Steps[1].Kind != StepMap || p.Steps[1].MapRef != "chapters" || p.Steps[1].MapMethod != "flesh-out" {
		t.Errorf("step 1: got %+v", p.Steps[1])
	}
}
