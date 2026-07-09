// Package digest provides targeted digest generation for Git repositories.
package digest

// DependencyDelta represents changes to Go module dependencies.
type DependencyDelta struct {
	Ecosystem            string
	SourceStatus         string
	GoModChanged         bool
	GoSumChanged         bool
	ModulePathChanged    bool
	GoVersionChanged     bool
	ToolchainChanged     bool
	RequiresAdded        int
	RequiresRemoved      int
	RequiresModified     int
	ReplacesAdded        int
	ReplacesRemoved      int
	ReplacesModified     int
	GoSumAdded           int
	GoSumRemoved         int
	Module               ModuleInfo
	GoVersion            VersionInfo
	Toolchain            VersionInfo
	RequiresAddedList    []string
	RequiresRemovedList  []string
	RequiresModifiedList []string
	ReplacesAddedList    []string
	ReplacesRemovedList  []string
	ReplacesModifiedList []string
	GoSumAddedList       []string
	GoSumRemovedList     []string
}

// ModuleInfo contains module directive changes.
type ModuleInfo struct {
	Before string
	After  string
}

// VersionInfo contains version directive changes.
type VersionInfo struct {
	Before string
	After  string
}
