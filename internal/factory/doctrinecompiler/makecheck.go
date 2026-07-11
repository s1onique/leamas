package doctrinecompiler

import (
	"fmt"
	"strings"
)

// verifyMakefileInclude asserts that the Makefile at path contains
// the generated factory fragment include.
func verifyMakefileInclude(resolver *Resolver, path TargetPath, contractID string) error {
	mkPath := resolver.Resolve(path)
	data, err := readFS(mkPath)
	if err != nil {
		return fmt.Errorf("contract %s: read %s: %v", contractID, path, err)
	}
	for _, line := range logicalLines(string(data)) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "include ") {
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "include "))
			rest = strings.TrimPrefix(rest, "./")
			if rest == ".factory/generated/factory.mk" {
				return nil
			}
		}
	}
	return fmt.Errorf("contract %s: Makefile at %s does not include .factory/generated/factory.mk", contractID, path)
}

// verifyMakefileTargetDep asserts that the Makefile at path declares
// a transitive (cycle-free) dependency chain from target to dep. It
// rejects any cycle encountered in the reachable subgraph.
//
// The traversal uses a three-state DFS (unvisited, active, complete)
// to detect arbitrary cycles, not just direct self-loops, and walks
// the full reachable graph so cycles in sibling branches are caught.
func verifyMakefileTargetDep(resolver *Resolver, path TargetPath, target, dep string) error {
	if target == "" || dep == "" {
		return fmt.Errorf("contract: empty target or dependency")
	}
	if target == dep {
		return fmt.Errorf("contract: target %q equals dependency %q (recursive cycle)", target, dep)
	}
	mkPath := resolver.Resolve(path)
	data, err := readFS(mkPath)
	if err != nil {
		return fmt.Errorf("read %s: %v", path, err)
	}
	deps := parseMakeDeps(strings.Join(logicalLines(string(data)), "\n"))
	if _, ok := deps[target]; !ok {
		return fmt.Errorf("contract: Makefile does not declare target %q", target)
	}
	reach, cycleTarget := makeReachability(deps, target)
	if !reach[dep] {
		return fmt.Errorf("contract: target %q has no dependency path to %q", target, dep)
	}
	if cycleTarget != "" {
		return fmt.Errorf("contract: target %q has a cycle reachable through %q", target, cycleTarget)
	}
	return nil
}

// logicalLines joins physical lines that end with a backslash
// continuation marker, mirroring GNU make's "Splitting Lines" rule.
//
// A backslash followed immediately by a newline joins the next
// physical line into the current logical line, with the trailing
// whitespace and the backslash removed. Comments inside logical
// lines are removed.
func logicalLines(content string) []string {
	rawLines := strings.Split(content, "\n")
	out := make([]string, 0, len(rawLines))
	var current strings.Builder
	joined := false
	for _, line := range rawLines {
		if joined {
			current.WriteString(" ")
		}
		// Look for trailing backslash (GNU make continuation).
		if strings.HasSuffix(line, "\\") {
			current.WriteString(strings.TrimSuffix(line, "\\"))
			joined = true
			continue
		}
		current.WriteString(line)
		out = append(out, current.String())
		current.Reset()
		joined = false
	}
	return out
}

// makeReachability performs a three-state DFS from root and returns
// the set of reachable targets plus the label of any cycle node
// encountered.
func makeReachability(deps map[string][]string, root string) (reachable map[string]bool, cycleTarget string) {
	reachable = make(map[string]bool)
	const (
		unvisited = 0
		active    = 1
		complete  = 2
	)
	state := make(map[string]int)
	var visit func(string) bool
	visit = func(n string) bool {
		switch state[n] {
		case active:
			cycleTarget = n
			return true
		case complete:
			return false
		}
		state[n] = active
		reachable[n] = true
		for _, d := range deps[n] {
			if visit(d) {
				return true
			}
		}
		state[n] = complete
		return false
	}
	visit(root)
	return reachable, cycleTarget
}

// parseMakeDeps returns a map from target name to its dependency
// list. The parser operates on logical lines (after backslash
// continuation joining) and ignores comments and blank lines.
func parseMakeDeps(content string) map[string][]string {
	out := make(map[string][]string)
	for _, line := range logicalLines(content) {
		// Strip trailing inline comment.
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, ":") {
			continue
		}
		idx := strings.Index(trimmed, ":")
		tgt := strings.TrimSpace(trimmed[:idx])
		rest := strings.TrimSpace(trimmed[idx+1:])
		if i := strings.IndexAny(rest, ";="); i >= 0 {
			rest = rest[:i]
		}
		if tgt == "" || strings.Contains(tgt, " ") || strings.ContainsAny(tgt, "%") {
			continue
		}
		if strings.HasPrefix(tgt, ".") {
			continue
		}
		for _, tok := range strings.Fields(rest) {
			out[tgt] = append(out[tgt], tok)
		}
	}
	return out
}
