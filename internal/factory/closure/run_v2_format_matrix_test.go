// SPDX-License-Identifier: Apache-2.0

package closure

import "testing"

func TestRunClosureV2ObjectFormatNEWAndRecoveryMatrix(t *testing.T) {
	for _, objectFormat := range supportedObjectFormats(t) {
		t.Run(string(objectFormat)+"/NEW", func(t *testing.T) {
			fixture := prepareV2RepositoryWithFormat(t, objectFormat)
			result, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, nil))
			if err != nil {
				t.Fatal(err)
			}
			assertCompleteV2Result(t, fixture, result)
			assertV2ResultObjectFormat(t, objectFormat, result)
		})

		t.Run(string(objectFormat)+"/recovery", func(t *testing.T) {
			fixture := prepareV2RepositoryWithFormat(t, objectFormat)
			checks := 0
			deps := productionV2TestDependencies(fixture, &v2FailingGit{failCommand: "update-ref"}, &checks)
			built := captureV2BuiltIdentities(&deps)
			if _, err := runProductionV2Test(fixture, deps); err == nil {
				t.Fatal("injected PREPARED interruption was accepted")
			}
			checks = 0
			result, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, &checks))
			if err != nil {
				t.Fatal(err)
			}
			assertCompleteV2Result(t, fixture, result)
			assertBuiltV2Identities(t, built, result)
			assertV2ResultObjectFormat(t, objectFormat, result)
			if checks != 0 {
				t.Fatalf("recovery checks = %d, want 0", checks)
			}
		})
	}
}

func prepareV2RepositoryWithFormat(t *testing.T, objectFormat ObjectFormat) v2RepositoryFixture {
	t.Helper()
	fixture := initializeV2RepositoryWithFormat(t, objectFormat)
	writeV2Plan(t, fixture)
	v2Git(t, fixture.root, "add", "docs/closure-plans")
	v2Git(t, fixture.root, "commit", "-m", "freeze plan")
	fixture.freeze = v2Git(t, fixture.root, "rev-parse", "HEAD")
	v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "subject")
	fixture.subject = v2Git(t, fixture.root, "rev-parse", "HEAD")
	return fixture
}

func assertV2ResultObjectFormat(t *testing.T, objectFormat ObjectFormat, result *TransactionResult) {
	t.Helper()
	for name, oid := range map[string]string{
		"F": result.FreezeCommit, "S": result.SubjectCommit, "C": result.ClosureCommit,
		"C_TREE": result.ClosureTree, "T": result.TagObject,
	} {
		if err := ValidateOIDWithFormat(name, oid, objectFormat); err != nil {
			t.Fatal(err)
		}
	}
}
