// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
)

// newProtectedDupcodeRunner returns an invocation-local dupcodeRunner. If
// depsFactory is non-nil (tests), the deps factory is used. In
// production the deps factory is nil and the binder constructs the
// real protected runner via protectedverifier.NewDupcodeRunner.
func newProtectedDupcodeRunner(depsFactory func() dupcodeRunner) dupcodeRunner {
	if depsFactory != nil {
		return depsFactory()
	}
	return &protectedDupcodeRunnerAdapter{inner: protectedverifier.NewDupcodeRunner()}
}

// protectedDupcodeRunnerAdapter adapts *protectedverifier.DupcodeRunner
// to the invocation-local dupcodeRunner interface. It is the only
// authorized location for the protected constructor call.
type protectedDupcodeRunnerAdapter struct {
	inner *protectedverifier.DupcodeRunner
}

func (a *protectedDupcodeRunnerAdapter) LoadBaseline(path string) (dupcode.Baseline, error) {
	return a.inner.LoadBaseline(path)
}

func (a *protectedDupcodeRunnerAdapter) RunCheckRepo(root string, cfg dupcode.Config) ([]dupcode.Finding, error) {
	return a.inner.RunCheckRepo(root, cfg)
}

func (a *protectedDupcodeRunnerAdapter) RunCheckReport(root string, cfg dupcode.Config) (dupcode.Report, error) {
	return a.inner.RunCheckReport(root, cfg)
}

func (a *protectedDupcodeRunnerAdapter) VerifyBaseline(root string, policy dupcode.BaselinePolicy) ([]checks.Finding, error) {
	return a.inner.VerifyBaseline(root, policy)
}

func (a *protectedDupcodeRunnerAdapter) WriteBaseline(path string, report dupcode.Report) error {
	return a.inner.WriteBaseline(path, report)
}

func (a *protectedDupcodeRunnerAdapter) CompareToBaseline(report dupcode.Report, baseline dupcode.Baseline) dupcode.CompareResult {
	return a.inner.CompareToBaseline(report, baseline)
}
