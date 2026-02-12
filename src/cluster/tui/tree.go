package tui

import (
	"fmt"
	"sort"
	"strings"

	"p2p/cluster"
)

func deriveTree(objects []cluster.ClusterObject, runs map[string]cluster.AgentRunSnapshot,
	pipelines map[string]*cluster.PipelineDef, search string,
	expanded map[string]bool) []Entry {

	sorted := make([]cluster.ClusterObject, len(objects))
	copy(sorted, objects)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	searchLower := strings.ToLower(search)
	var entries []Entry

	for _, obj := range sorted {
		if search != "" && !strings.Contains(strings.ToLower(obj.Name), searchLower) {
			continue
		}

		agentKey := entryKey(obj.Name, "")
		agentExp := isExpanded(expanded, agentKey)
		run, hasRun := runs[obj.Name]
		pdef := pipelines[obj.Name]

		entries = append(entries, Entry{
			Kind: NodeAgent, Label: obj.Name, Agent: obj.Name,
			HasChildren: true, Expanded: agentExp,
		})
		if !agentExp {
			continue
		}

		if pdef != nil && len(pdef.Steps) > 0 {
			for _, step := range pdef.Steps {
				var label, stepLabel string
				switch step.Kind {
				case cluster.StepKindLoop:
					stepLabel = step.LoopMethod
					label = fmt.Sprintf("loop(%s)", stepLabel)
				case cluster.StepKindMap:
					stepLabel = step.MapMethod
					label = fmt.Sprintf("map(%s)", stepLabel)
				case cluster.StepKindSimple:
					stepLabel = step.Method
					label = step.Label
				default:
					stepLabel = step.Label
					label = step.Label
				}

				stepKey := entryKey(obj.Name, stepLabel)
				hasIters := step.Kind == cluster.StepKindLoop && hasRun &&
					(run.LiveIter != nil || len(run.Iterations) > 0)

				entries = append(entries, Entry{
					Kind: NodeLoop, Label: label, Agent: obj.Name, Step: stepLabel,
					Depth: 1, HasChildren: hasIters, Expanded: isExpanded(expanded, stepKey),
				})
				if isExpanded(expanded, stepKey) && hasIters {
					appendIters(&entries, obj.Name, stepLabel, run)
				}
			}
		} else {
			stepLabel := extractLoopMethod(obj.Definition)
			if stepLabel == "" {
				stepLabel = "loop"
			}
			stepKey := entryKey(obj.Name, stepLabel)
			hasIters := hasRun && (run.LiveIter != nil || len(run.Iterations) > 0)

			entries = append(entries, Entry{
				Kind: NodeLoop, Label: fmt.Sprintf("loop(%s)", stepLabel),
				Agent: obj.Name, Step: stepLabel, Depth: 1,
				HasChildren: hasIters, Expanded: isExpanded(expanded, stepKey),
			})
			if isExpanded(expanded, stepKey) && hasIters {
				appendIters(&entries, obj.Name, stepLabel, run)
			}
		}
	}
	return entries
}

func appendIters(entries *[]Entry, agent, step string, run cluster.AgentRunSnapshot) {
	if run.LiveIter != nil {
		*entries = append(*entries, Entry{
			Kind: NodeIteration, Label: fmt.Sprintf("iteration %d (live)", run.LiveIter.Iteration),
			Agent: agent, Step: step, Iter: run.LiveIter.Iteration, Depth: 2, Live: true,
		})
	}
	if len(run.Iterations) > 0 {
		iters := run.Iterations
		start := 0
		if len(iters) > 4 {
			start = len(iters) - 4
		}
		for i := len(iters) - 1; i >= start; i-- {
			*entries = append(*entries, Entry{
				Kind: NodeIteration, Label: fmt.Sprintf("iteration %d", iters[i].Iteration),
				Agent: agent, Step: step, Iter: iters[i].Iteration, Depth: 2,
			})
		}
	}
}
