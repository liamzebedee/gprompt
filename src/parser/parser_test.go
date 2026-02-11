package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParseRalph(t *testing.T) {
	path := "../../examples/ralph/ralph.p"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("ralph.p not found at %s", path)
	}

	nodes, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var methods []string
	var invocations []string
	for _, n := range nodes {
		switch n.Type {
		case NodeMethodDef:
			methods = append(methods, n.Name)
		case NodeInvocation:
			invocations = append(invocations, n.Name)
		}
	}

	expectedMethods := []string{
		"simpleeverything", "ralph", "supervise-build",
		"spec", "plan", "build", "jtbd", "reportbug", "bugfix",
	}
	for _, want := range expectedMethods {
		found := false
		for _, got := range methods {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing method definition: %q (got methods: %v)", want, methods)
		}
	}

	expectedInvocations := []string{"simpleeverything", "ralph"}
	for _, want := range expectedInvocations {
		found := false
		for _, got := range invocations {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing invocation: %q (got invocations: %v)", want, invocations)
		}
	}

	for _, n := range nodes {
		if n.Type == NodeMethodDef && n.Name == "ralph" {
			if n.Body == "" {
				t.Error("ralph method has empty body")
			}
			if !strings.Contains(n.Body, " -> ") {
				t.Errorf("ralph body should be a pipeline, got: %q", n.Body)
			}
			return
		}
	}
	t.Error("ralph method not found")
}

func TestParseRalphPipeline(t *testing.T) {
	input := "ralph(idea):\n\tidea -> spec -> plan -> loop(build)\n"
	nodes, err := ParseString(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	for _, n := range nodes {
		if n.Type == NodeMethodDef && n.Name == "ralph" {
			if n.Body != "idea -> spec -> plan -> loop(build)" {
				t.Errorf("unexpected body: %q", n.Body)
			}
			return
		}
	}
	t.Error("ralph method not found")
}

func TestParseBook(t *testing.T) {
	path := "../../examples/book/book.p"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("book.p not found at %s", path)
	}

	nodes, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var methods []string
	var invocations []string
	for _, n := range nodes {
		switch n.Type {
		case NodeMethodDef:
			methods = append(methods, n.Name)
		case NodeInvocation:
			invocations = append(invocations, n.Name)
		}
	}

	expectedMethods := []string{
		"book", "book-idea", "generate-chapter-index",
		"flesh-out-chapter", "concat",
	}
	for _, want := range expectedMethods {
		found := false
		for _, got := range methods {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing method definition: %q (got methods: %v)", want, methods)
		}
	}

	if len(invocations) != 1 || invocations[0] != "book" {
		t.Errorf("expected invocation [@book], got %v", invocations)
	}

	for _, n := range nodes {
		if n.Type == NodeMethodDef && n.Name == "book" {
			if !strings.Contains(n.Body, " -> ") {
				t.Errorf("book body should be a pipeline, got: %q", n.Body)
			}
			return
		}
	}
	t.Error("book method not found")
}

func TestParsePipeline(t *testing.T) {
	input := "mypipe(x):\n\tx -> step1 (foo) -> step2 (bar)\n"
	nodes, err := ParseString(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	for _, n := range nodes {
		if n.Type == NodeMethodDef && n.Name == "mypipe" {
			if !strings.Contains(n.Body, " -> ") {
				t.Errorf("pipeline body not captured, got: %q", n.Body)
			}
			return
		}
	}
	t.Error("mypipe method not found")
}
