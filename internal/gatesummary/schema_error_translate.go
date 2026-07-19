package gatesummary

import (
	"reflect"
	"sort"
	"strconv"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// schemaErrorTranslator converts structured jsonschema/v6 error trees into
// stable diagnostics without rendering validator messages.
type schemaErrorTranslator struct {
	root *jsonschema.ValidationError
}

func (t schemaErrorTranslator) translate() []Diagnostic {
	ds := &diagnosticSet{}
	if !validRootWrapper(t.root) {
		ds.add(newDiagnostic(CodeInternal, "/", "malformed schema validation root"))
		return ds.emit()
	}
	t.walkCauses(t.root.Causes, ds)
	return ds.emit()
}

func validRootWrapper(root *jsonschema.ValidationError) bool {
	if root == nil || len(root.Causes) == 0 {
		return false
	}
	rootKind, ok := root.ErrorKind.(*kind.Schema)
	return ok && rootKind != nil
}

// walkCauses uses an explicit stack so hostile validator nesting cannot
// exhaust the goroutine stack. Reverse pushes preserve cause-slice order.
func (t schemaErrorTranslator) walkCauses(
	causes []*jsonschema.ValidationError,
	ds *diagnosticSet,
) {
	stack := make([]*jsonschema.ValidationError, 0, len(causes))
	for i := len(causes) - 1; i >= 0; i-- {
		stack = append(stack, causes[i])
	}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		if node == nil || isNilErrorKind(node.ErrorKind) {
			ds.add(newDiagnostic(CodeInternal, "/", "malformed schema validation cause"))
			continue
		}
		if isWrapperKind(node.ErrorKind) {
			if len(node.Causes) == 0 {
				ds.add(newDiagnostic(CodeSchemaViolation,
					instanceLocationToPointer(node.InstanceLocation), "empty wrapper"))
				continue
			}
			for i := len(node.Causes) - 1; i >= 0; i-- {
				stack = append(stack, node.Causes[i])
			}
			continue
		}
		t.translateNode(node, ds)
	}
}

func isNilErrorKind(errorKind jsonschema.ErrorKind) bool {
	if errorKind == nil {
		return true
	}
	value := reflect.ValueOf(errorKind)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

func isWrapperKind(errorKind jsonschema.ErrorKind) bool {
	switch k := errorKind.(type) {
	case *kind.Schema:
		return k != nil
	case *kind.Group:
		return k != nil
	case *kind.Reference:
		return k != nil
	default:
		return false
	}
}

func (t schemaErrorTranslator) translateNode(
	node *jsonschema.ValidationError,
	ds *diagnosticSet,
) {
	location := node.InstanceLocation
	path := instanceLocationToPointer(location)
	baseURL, keywordTokens := keywordIdentity(node)

	switch k := node.ErrorKind.(type) {
	case *kind.Required:
		if containsString(k.Missing, "schema_version") {
			ds.add(newDiagnostic(CodeInternal, "/schema_version",
				"schema_version missing after dispatch"))
			return
		}
		t.fanoutMissing(ds, path, k.Missing)
		return
	case *kind.AdditionalProperties:
		t.fanoutAdditional(ds, path, k.Properties)
		return
	}

	if _, ok := node.ErrorKind.(*kind.AnyOf); ok &&
		isTestTotalAnyOf(baseURL, keywordTokens) {
		ds.add(newDiagnostic(CodePartialTestTotals, path,
			"test totals must all be present or all be absent"))
		return
	}

	if isSchemaVersionLocation(location) {
		switch node.ErrorKind.(type) {
		case *kind.Type, *kind.Const:
			ds.add(newDiagnostic(CodeInternal, "/schema_version",
				"schema_version mismatch after dispatch"))
			return
		}
	}

	switch k := node.ErrorKind.(type) {
	case *kind.Enum:
		t.emitEnum(ds, location, path)
	case *kind.Type:
		t.emitType(ds, location, path)
	case *kind.Pattern:
		t.emitPattern(ds, location, path)
	case *kind.Minimum:
		if isDurationLocation(location) {
			ds.add(newDiagnostic(CodeInvalidDuration, path, "duration_ms is negative"))
		} else {
			ds.add(newDiagnostic(CodeSchemaViolation, path, "minimum violation"))
		}
	case *kind.Format, *kind.MinLength:
		if isGeneratedAtLocation(location) {
			ds.add(newDiagnostic(CodeInvalidTimestamp, "/generated_at",
				"generated_at must be a valid RFC 3339 timestamp"))
		} else {
			ds.add(newDiagnostic(CodeSchemaViolation, path,
				"format or min-length violation"))
		}
	case *kind.MaxItems:
		t.emitMaxItems(ds, baseURL, path, k)
	case *kind.MaxLength:
		t.emitMaxLength(ds, baseURL, path, k)
	default:
		ds.add(newDiagnostic(CodeSchemaViolation, path, "unmapped selected-schema leaf"))
	}
}

func (t schemaErrorTranslator) emitEnum(ds *diagnosticSet, location []string, path string) {
	if isStatusLocation(location) {
		ds.add(newDiagnostic(CodeInvalidStatus, path, "invalid status value"))
		return
	}
	ds.add(newDiagnostic(CodeSchemaViolation, path, "enum mismatch"))
}

func (t schemaErrorTranslator) emitType(ds *diagnosticSet, location []string, path string) {
	if isOIDLocation(location) {
		ds.add(newDiagnostic(CodeInvalidOID, path, "invalid execution OID"))
		return
	}
	ds.add(newDiagnostic(CodeSchemaViolation, path, "type mismatch"))
}

func (t schemaErrorTranslator) emitPattern(ds *diagnosticSet, location []string, path string) {
	switch {
	case isOIDLocation(location):
		ds.add(newDiagnostic(CodeInvalidOID, path, "invalid execution OID"))
	case isOutputHashLocation(location):
		ds.add(newDiagnostic(CodeInvalidOutputHash, path,
			"output hash is not 64 lowercase hex"))
	default:
		ds.add(newDiagnostic(CodeSchemaViolation, path, "pattern mismatch"))
	}
}

func (t schemaErrorTranslator) emitMaxItems(
	ds *diagnosticSet,
	baseURL string,
	path string,
	k *kind.MaxItems,
) {
	if baseURL != v2SchemaID {
		ds.add(newDiagnostic(CodeSchemaViolation, path, "max-items violation"))
		return
	}
	d := newDiagnostic(CodeCollectionLimit, path, "collection exceeds schema-defined maximum")
	d.Expected = strconv.Itoa(k.Want)
	d.Observed = strconv.Itoa(k.Got)
	ds.add(d)
}

func (t schemaErrorTranslator) emitMaxLength(
	ds *diagnosticSet,
	baseURL string,
	path string,
	k *kind.MaxLength,
) {
	if baseURL != v2SchemaID {
		ds.add(newDiagnostic(CodeSchemaViolation, path, "max-length violation"))
		return
	}
	d := newDiagnostic(CodeCollectionLimit, path, "string exceeds schema-defined maximum")
	d.Expected = strconv.Itoa(k.Want)
	d.Observed = strconv.Itoa(k.Got)
	ds.add(d)
}

func (t schemaErrorTranslator) fanoutMissing(
	ds *diagnosticSet,
	instancePath string,
	missing []string,
) {
	names := append([]string(nil), missing...)
	sort.Strings(names)
	for _, name := range names {
		ds.add(newDiagnostic(CodeRequiredFieldMissing,
			appendPointer(instancePath, name), "required field is missing"))
	}
}

func (t schemaErrorTranslator) fanoutAdditional(
	ds *diagnosticSet,
	instancePath string,
	properties []string,
) {
	names := append([]string(nil), properties...)
	sort.Strings(names)
	for _, name := range names {
		ds.add(newDiagnostic(CodeUnknownField, appendPointer(instancePath, name),
			"unknown field rejected by additionalProperties:false"))
	}
}

func appendPointer(base, token string) string {
	return base + "/" + escapePointer(token)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
