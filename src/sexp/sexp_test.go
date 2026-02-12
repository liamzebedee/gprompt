package sexp

import (
	"strings"
	"testing"

	"p2p/parser"
	"p2p/registry"
)

func parseAndEmit(t *testing.T, source string, filter string) string {
	t.Helper()
	nodes, err := parser.ParseString(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	reg := registry.New()
	for _, n := range nodes {
		if n.Type == parser.NodeMethodDef {
			reg.Register(n.Name, n.Params, n.Body)
		}
	}
	return EmitProgram(nodes, reg, filter)
}

func TestYProgram(t *testing.T) {
	source := "@conversational\nhow do trees grow?\n@listify(n=10)\n"
	output := parseAndEmit(t, source, "")

	// Check program wrapper
	if !strings.HasPrefix(output, "(program\n") {
		t.Errorf("expected (program wrapper, got: %s", output[:40])
	}

	// Check forms present
	if !strings.Contains(output, "(invoke conversational)") {
		t.Error("missing (invoke conversational)")
	}
	if !strings.Contains(output, `(text "how do trees grow?")`) {
		t.Error("missing (text)")
	}
	if !strings.Contains(output, `(invoke listify :n "10")`) {
		t.Error("missing (invoke listify :n \"10\")")
	}
}

func TestBookPipeline(t *testing.T) {
	source := "book(topic):\n\ttopic -> brief (book-idea) -> chapter-outline (generate-chapter-index) -> chapters (map(chapters, flesh-out-chapter)) -> final (concat)\n\nbook-idea(topic):\n\tWe are writing a book about [topic].\n\n@book(blockchain)\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, "(defpipeline book (topic)") {
		t.Error("missing defpipeline book")
	}
	if !strings.Contains(output, `(step "brief" (call book-idea))`) {
		t.Error("missing step brief")
	}
	if !strings.Contains(output, `(step "chapters" (map chapters flesh-out-chapter))`) {
		t.Error("missing map step")
	}
	if !strings.Contains(output, `(invoke book "blockchain")`) {
		t.Error("missing invoke book")
	}
	if !strings.Contains(output, "(defmethod book-idea (topic)") {
		t.Error("missing defmethod book-idea")
	}
}

func TestJokerLoop(t *testing.T) {
	source := "joker:\n\tloop(joke)\n\njoke:\n\tTell a knock-knock joke.\n\n@joker\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, "(defpipeline joker ()") {
		t.Error("missing defpipeline joker")
	}
	if !strings.Contains(output, `(step "joke" (loop joke))`) {
		t.Error("missing loop step")
	}
	if !strings.Contains(output, "(defmethod joke ()") {
		t.Error("missing defmethod joke")
	}
	if !strings.Contains(output, "(invoke joker)") {
		t.Error("missing invoke joker")
	}
}

func TestAgents(t *testing.T) {
	source := "build:\n\tDo the build.\n\nagent-builder:\n\tloop(build)\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, "(defmethod build ()") {
		t.Error("missing defmethod build")
	}
	if !strings.Contains(output, `(defagent "builder"`) {
		t.Error("missing defagent builder")
	}
	if !strings.Contains(output, `(step "build" (loop build))`) {
		t.Error("missing loop step in agent")
	}
}

func TestIDCommentsPresent(t *testing.T) {
	source := "foo:\n\tdo stuff\n\n@foo\n"
	output := parseAndEmit(t, source, "")

	count := strings.Count(output, "; id=")
	if count != 2 {
		t.Errorf("expected 2 id comments, got %d", count)
	}
}

func TestIDCommentsStable(t *testing.T) {
	source := "foo:\n\tdo stuff\n\n@foo\n"
	output1 := parseAndEmit(t, source, "")
	output2 := parseAndEmit(t, source, "")

	if output1 != output2 {
		t.Error("id comments are not stable across runs")
	}
}

func TestFilterMode(t *testing.T) {
	source := "foo:\n\tdo foo\n\nbar:\n\tdo bar\n\n@foo\n"
	output := parseAndEmit(t, source, "foo")

	if strings.Contains(output, "(program") {
		t.Error("filter mode should not wrap in (program)")
	}
	if !strings.Contains(output, "(defmethod foo ()") {
		t.Error("missing filtered defmethod foo")
	}
	if strings.Contains(output, "bar") {
		t.Error("filter mode should not include bar")
	}
	if !strings.Contains(output, "; id=") {
		t.Error("filter mode should include id comment")
	}
}

func TestFilterModeNotFound(t *testing.T) {
	source := "foo:\n\tdo foo\n"
	output := parseAndEmit(t, source, "nonexistent")

	if output != "" {
		t.Errorf("expected empty output for nonexistent filter, got: %s", output)
	}
}

func TestImport(t *testing.T) {
	source := "@utils.p\n@foo\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, `(import "utils.p")`) {
		t.Error("missing import form")
	}
}

func TestPlainText(t *testing.T) {
	source := "hello world\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, `(text "hello world")`) {
		t.Error("missing text form")
	}
}

func TestMultilineBody(t *testing.T) {
	source := "foo:\n\tline one\n\tline two\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, `"line one\nline two"`) {
		t.Error("multiline body should use \\n")
	}
}

func TestKeywordArgs(t *testing.T) {
	source := "@method(key1=val1, key2=val2)\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, `:key1 "val1"`) {
		t.Error("missing keyword arg key1")
	}
	if !strings.Contains(output, `:key2 "val2"`) {
		t.Error("missing keyword arg key2")
	}
}

func TestTrailingText(t *testing.T) {
	source := "@method some trailing text\n"
	output := parseAndEmit(t, source, "")

	if !strings.Contains(output, `:trailing "some trailing text"`) {
		t.Error("missing trailing text")
	}
}
