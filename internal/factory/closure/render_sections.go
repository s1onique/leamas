package closure

import (
	"fmt"
	"strings"
)

func renderPolicySection(report *strings.Builder, manifest Manifest) {
	report.WriteString("## Patch hygiene\n\n")
	fmt.Fprintf(report, "- Git diff check: %s\n", strings.ToUpper(manifest.PatchHygiene.Status))
	fmt.Fprintf(report, "- Diagnostics: %d\n", manifest.PatchHygiene.DiagnosticCount)
	fmt.Fprintf(report, "- Tracked full digest policy: %s\n", strings.ToUpper(manifest.ClosurePolicy.TrackedFullDigestStatus))
	fmt.Fprintf(report, "- Closure-policy diagnostics: %d\n\n", manifest.ClosurePolicy.DiagnosticCount)
}

func renderRunnerSection(report *strings.Builder, runner RunnerIdentity) {
	report.WriteString("## Runner identity\n\n")
	fmt.Fprintf(report, "- Leamas version: `%s`\n", runner.LeamasVersion)
	fmt.Fprintf(report, "- Binary SHA-256: `%s`\n", runner.BinarySHA256)
	fmt.Fprintf(report, "- VCS revision: `%s`\n", runner.VCSRevision)
	fmt.Fprintf(report, "- VCS modified: `%t`\n\n", runner.VCSModified)
}

func renderLifecycleSection(report *strings.Builder, manifest Manifest) {
	report.WriteString("## Lifecycle transition\n\n")
	state := LifecycleImplemented
	if manifest.Verdict == VerdictPass {
		state = LifecycleVerified
	}
	fmt.Fprintf(report, "Verification state: %s\n\n", state)
	report.WriteString("The immutable closure tag is created after this report and manifest are committed. ")
	report.WriteString("The annotated-tag object identity remains external Git evidence.\n")
}

func joinRepositoryPath(root, slashPath string) string {
	if root == "." {
		return slashPath
	}
	return root + "/" + slashPath
}
