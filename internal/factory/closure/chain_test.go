package closure

import "testing"

func TestRejectPlaceholder_TODO(t *testing.T) {
	err := RejectPlaceholder("test", "TODO")
	if err == nil {
		t.Error("expected rejection of TODO")
	}
}

func TestRejectPlaceholder_TBD(t *testing.T) {
	err := RejectPlaceholder("test", "TBD")
	if err == nil {
		t.Error("expected rejection of TBD")
	}
}

func TestRejectPlaceholder_UNKNOWN(t *testing.T) {
	err := RejectPlaceholder("test", "UNKNOWN")
	if err == nil {
		t.Error("expected rejection of UNKNOWN")
	}
}

func TestRejectPlaceholder_PlaceholderGitOID(t *testing.T) {
	for _, val := range []string{"a1b2c3d4e5f6", "deadbeef"} {
		err := RejectPlaceholder("test", val)
		if err == nil {
			t.Errorf("expected rejection of %q", val)
		}
	}
}

func TestRejectPlaceholder_EmbeddedPlaceholder(t *testing.T) {
	for _, val := range []string{"<COMMIT>", "<TREE>", "<HASH>", "(SEE GIT REV-PARSE)"} {
		err := RejectPlaceholder("test", val)
		if err == nil {
			t.Errorf("expected rejection of %q", val)
		}
	}
}

func TestRejectPlaceholder_CaseInsensitive(t *testing.T) {
	err := RejectPlaceholder("test", "todo")
	if err == nil {
		t.Error("expected rejection of lowercase todo")
	}
}

func TestRejectPlaceholder_ValidOID(t *testing.T) {
	err := RejectPlaceholder("test", "8362d35c65f66ccd140f5b5044b776f435fdc711")
	if err != nil {
		t.Errorf("unexpected rejection of valid OID: %v", err)
	}
}

func TestRejectPlaceholder_Empty(t *testing.T) {
	err := RejectPlaceholder("test", "")
	if err != nil {
		t.Errorf("empty value should be allowed: %v", err)
	}
}

func TestValidateOID_Valid(t *testing.T) {
	err := ValidateOID("commit", "8362d35c65f66ccd140f5b5044b776f435fdc711")
	if err != nil {
		t.Errorf("expected valid OID: %v", err)
	}
}

func TestValidateOID_InvalidLength(t *testing.T) {
	err := ValidateOID("commit", "8362d35c65f66ccd140f5b5044b776f435fdc7")
	if err == nil {
		t.Error("expected rejection of truncated OID")
	}
}

func TestValidateOID_Placeholder(t *testing.T) {
	err := ValidateOID("commit", "TODO")
	if err == nil {
		t.Error("expected rejection of placeholder")
	}
}

func TestValidateOID_WrongFormat(t *testing.T) {
	err := ValidateOID("commit", "not-a-hex-string")
	if err == nil {
		t.Error("expected rejection of non-hex string")
	}
}
