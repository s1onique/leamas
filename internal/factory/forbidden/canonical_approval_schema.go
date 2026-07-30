// SPDX-License-Identifier: Apache-2.0

package forbidden

import "strings"

// approvalSchemaIssue is one finding emitted by validateApprovalSchema.
//
// The helper is pure: it never mutates the input approval and returns issues
// in a deterministic fixed field order so callers, tests, and reporters can
// rely on a stable sequence:
//
//	PackagePath
//	Function
//	CallerKind
//	Receiver
//	ReferenceClass
//	Cardinality
//	Callee.Layer
//	Callee.PackagePath
//	Callee.Name
//	Callee.Kind
//	Callee.Receiver
type approvalSchemaIssue struct {
	Kind    string
	Field   string
	Message string
}

const (
	approvalIdentityInvalidKind       = "authority_policy_approval_identity_invalid"
	approvalCallerKindInvalidKind     = "authority_policy_approval_caller_kind_invalid"
	approvalReceiverInvalidKind       = "authority_policy_approval_receiver_invalid"
	approvalReferenceClassInvalidKind = "authority_policy_approval_reference_class_invalid"
	approvalCardinalityInvalidKind    = "authority_policy_approval_cardinality_invalid"
	approvalCalleeLayerInvalidKind    = "authority_policy_approval_callee_layer_invalid"
	approvalCalleeIdentityInvalidKind = "authority_policy_approval_callee_identity_invalid"
	approvalCalleeKindInvalidKind     = "authority_policy_approval_callee_kind_invalid"
	approvalCalleeReceiverInvalidKind = "authority_policy_approval_callee_receiver_invalid"
)

const (
	fieldPackagePath       = "PackagePath"
	fieldFunction          = "Function"
	fieldCallerKind        = "CallerKind"
	fieldReceiver          = "Receiver"
	fieldReferenceClass    = "ReferenceClass"
	fieldCardinality       = "Cardinality"
	fieldCalleeLayer       = "Callee.Layer"
	fieldCalleePackagePath = "Callee.PackagePath"
	fieldCalleeName        = "Callee.Name"
	fieldCalleeKind        = "Callee.Kind"
	fieldCalleeReceiver    = "Callee.Receiver"
)

// validateApprovalSchema returns a deterministic, ordered list of schema
// issues for the supplied approval. The input is never mutated or normalized.
// An empty result means the approval passes strict schema validation and may
// participate in duplicate detection, caller/callee resolution, observed-edge
// matching, stale-approval checking, and cardinality checking.
//
// Caller-side rules:
//   - PackagePath must be non-empty and contain no wildcards.
//   - Function must be non-empty, contain no wildcards, and contain no
//     source-coordinate markers ("@").
//   - CallerKind must be one of package_function, method,
//     variable_initializer, or package_init. Empty, unknown values, and the
//     function_literal kind are rejected.
//   - Receiver must be non-empty and wildcard-free for methods; empty for
//     package_function, variable_initializer, and package_init.
//   - ReferenceClass must be one of DIRECT_CALL, FUNCTION_VALUE,
//     METHOD_VALUE, METHOD_EXPRESSION, or PACKAGE_VARIABLE_REFERENCE.
//     DOT_IMPORT and DECLARATION are observed by traversal but never
//     approvable.
//   - Cardinality must be strictly positive.
//
// Callee-side rules:
//   - Callee.Layer must be one of raw_dupcode, protected_adapter, or
//     factorize_gate.
//   - Callee.PackagePath must be non-empty and contain no wildcards.
//   - Callee.Name must be non-empty, contain no wildcards, and contain no
//     source-coordinate markers.
//   - Callee.Kind must be one of package_function, method, or
//     package_variable.
//   - Callee.Receiver must be non-empty and wildcard-free for methods;
//     empty for package_function and package_variable.
func validateApprovalSchema(approval ApprovedCaller) []approvalSchemaIssue {
	var issues []approvalSchemaIssue

	issues = appendIssueForPackagePath(issues, approval.PackagePath)
	issues = appendIssueForFunction(issues, approval.Function)
	issues = appendIssueForCallerKind(issues, approval.CallerKind)
	issues = appendIssueForReceiver(issues, approval.CallerKind, approval.Receiver)
	issues = appendIssueForReferenceClass(issues, approval.ReferenceClass)
	issues = appendIssueForCardinality(issues, approval.Cardinality)

	issues = appendIssueForCalleeLayer(issues, approval.Callee.Layer)
	issues = appendIssueForCalleePackagePath(issues, approval.Callee.PackagePath)
	issues = appendIssueForCalleeName(issues, approval.Callee.Name)
	issues = appendIssueForCalleeKind(issues, approval.Callee.Kind)
	issues = appendIssueForCalleeReceiver(issues, approval.Callee.Kind, approval.Callee.Receiver)

	return issues
}

func appendIssueForPackagePath(issues []approvalSchemaIssue, value string) []approvalSchemaIssue {
	switch {
	case value == "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalIdentityInvalidKind,
			Field:   fieldPackagePath,
			Message: "package_path_empty",
		})
	case strings.Contains(value, "*"):
		return append(issues, approvalSchemaIssue{
			Kind:    approvalIdentityInvalidKind,
			Field:   fieldPackagePath,
			Message: "package_path_wildcard",
		})
	}
	return issues
}

func appendIssueForFunction(issues []approvalSchemaIssue, value string) []approvalSchemaIssue {
	switch {
	case value == "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalIdentityInvalidKind,
			Field:   fieldFunction,
			Message: "function_empty",
		})
	case strings.Contains(value, "@"):
		return append(issues, approvalSchemaIssue{
			Kind:    approvalIdentityInvalidKind,
			Field:   fieldFunction,
			Message: "function_source_coordinate",
		})
	case strings.Contains(value, "*"):
		return append(issues, approvalSchemaIssue{
			Kind:    approvalIdentityInvalidKind,
			Field:   fieldFunction,
			Message: "function_wildcard",
		})
	}
	return issues
}

func appendIssueForCallerKind(issues []approvalSchemaIssue, kind string) []approvalSchemaIssue {
	switch kind {
	case "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCallerKindInvalidKind,
			Field:   fieldCallerKind,
			Message: "caller_kind_empty",
		})
	case CallerKindPackageFunction, CallerKindMethod,
		CallerKindVariableInitializer, CallerKindPackageInit:
		return issues
	default:
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCallerKindInvalidKind,
			Field:   fieldCallerKind,
			Message: "caller_kind_unknown",
		})
	}
}

func appendIssueForReceiver(issues []approvalSchemaIssue, kind, receiver string) []approvalSchemaIssue {
	switch kind {
	case CallerKindMethod:
		switch {
		case receiver == "":
			return append(issues, approvalSchemaIssue{
				Kind:    approvalReceiverInvalidKind,
				Field:   fieldReceiver,
				Message: "receiver_required_for_method",
			})
		case strings.Contains(receiver, "*"):
			return append(issues, approvalSchemaIssue{
				Kind:    approvalReceiverInvalidKind,
				Field:   fieldReceiver,
				Message: "receiver_wildcard_for_method",
			})
		}
	case CallerKindPackageFunction, CallerKindVariableInitializer, CallerKindPackageInit:
		if receiver != "" {
			return append(issues, approvalSchemaIssue{
				Kind:    approvalReceiverInvalidKind,
				Field:   fieldReceiver,
				Message: "receiver_must_be_empty",
			})
		}
	}
	return issues
}

func appendIssueForReferenceClass(issues []approvalSchemaIssue, class ReferenceClass) []approvalSchemaIssue {
	switch class {
	case "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalReferenceClassInvalidKind,
			Field:   fieldReferenceClass,
			Message: "reference_class_empty",
		})
	case refDotImport:
		return append(issues, approvalSchemaIssue{
			Kind:    approvalReferenceClassInvalidKind,
			Field:   fieldReferenceClass,
			Message: "reference_class_dot_import",
		})
	case refDeclaration:
		return append(issues, approvalSchemaIssue{
			Kind:    approvalReferenceClassInvalidKind,
			Field:   fieldReferenceClass,
			Message: "reference_class_declaration",
		})
	case refDirectCall, refFunctionValue, refMethodValue, refMethodExpression, refPackageVariable:
		return issues
	default:
		return append(issues, approvalSchemaIssue{
			Kind:    approvalReferenceClassInvalidKind,
			Field:   fieldReferenceClass,
			Message: "reference_class_unknown",
		})
	}
}

func appendIssueForCardinality(issues []approvalSchemaIssue, cardinality int) []approvalSchemaIssue {
	switch {
	case cardinality == 0:
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCardinalityInvalidKind,
			Field:   fieldCardinality,
			Message: "cardinality_zero",
		})
	case cardinality < 0:
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCardinalityInvalidKind,
			Field:   fieldCardinality,
			Message: "cardinality_negative",
		})
	}
	return issues
}

func appendIssueForCalleeLayer(issues []approvalSchemaIssue, layer AuthorityLayer) []approvalSchemaIssue {
	switch {
	case layer == "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeLayerInvalidKind,
			Field:   fieldCalleeLayer,
			Message: "callee_layer_empty",
		})
	case !validAuthorityLayer(layer):
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeLayerInvalidKind,
			Field:   fieldCalleeLayer,
			Message: "callee_layer_unknown",
		})
	}
	return issues
}

func appendIssueForCalleePackagePath(issues []approvalSchemaIssue, value string) []approvalSchemaIssue {
	switch {
	case value == "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeIdentityInvalidKind,
			Field:   fieldCalleePackagePath,
			Message: "callee_package_path_empty",
		})
	case strings.Contains(value, "*"):
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeIdentityInvalidKind,
			Field:   fieldCalleePackagePath,
			Message: "callee_package_path_wildcard",
		})
	}
	return issues
}

func appendIssueForCalleeName(issues []approvalSchemaIssue, value string) []approvalSchemaIssue {
	switch {
	case value == "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeIdentityInvalidKind,
			Field:   fieldCalleeName,
			Message: "callee_name_empty",
		})
	case strings.Contains(value, "@"):
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeIdentityInvalidKind,
			Field:   fieldCalleeName,
			Message: "callee_name_source_coordinate",
		})
	case strings.Contains(value, "*"):
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeIdentityInvalidKind,
			Field:   fieldCalleeName,
			Message: "callee_name_wildcard",
		})
	}
	return issues
}

func appendIssueForCalleeKind(issues []approvalSchemaIssue, kind ProtectedSymbolKind) []approvalSchemaIssue {
	switch kind {
	case "":
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeKindInvalidKind,
			Field:   fieldCalleeKind,
			Message: "callee_kind_empty",
		})
	case ProtectedPackageFunction, ProtectedMethod, ProtectedPackageVariable:
		return issues
	default:
		return append(issues, approvalSchemaIssue{
			Kind:    approvalCalleeKindInvalidKind,
			Field:   fieldCalleeKind,
			Message: "callee_kind_unknown",
		})
	}
}

func appendIssueForCalleeReceiver(issues []approvalSchemaIssue, kind ProtectedSymbolKind, receiver string) []approvalSchemaIssue {
	switch kind {
	case ProtectedMethod:
		switch {
		case receiver == "":
			return append(issues, approvalSchemaIssue{
				Kind:    approvalCalleeReceiverInvalidKind,
				Field:   fieldCalleeReceiver,
				Message: "callee_receiver_required_for_method",
			})
		case strings.Contains(receiver, "*"):
			return append(issues, approvalSchemaIssue{
				Kind:    approvalCalleeReceiverInvalidKind,
				Field:   fieldCalleeReceiver,
				Message: "callee_receiver_wildcard_for_method",
			})
		}
	case ProtectedPackageFunction, ProtectedPackageVariable:
		if receiver != "" {
			return append(issues, approvalSchemaIssue{
				Kind:    approvalCalleeReceiverInvalidKind,
				Field:   fieldCalleeReceiver,
				Message: "callee_receiver_must_be_empty",
			})
		}
	}
	return issues
}

// validApprovalReferenceClass reports whether a class may appear on a
// configured approval. DOT_IMPORT and DECLARATION are observed by traversal
// but are never approvable.
func validApprovalReferenceClass(class ReferenceClass) bool {
	switch class {
	case refDirectCall, refFunctionValue, refMethodValue, refMethodExpression, refPackageVariable:
		return true
	default:
		return false
	}
}

// validObservedReferenceClass reports whether a class may appear on an
// observed edge. DOT_IMPORT is intentionally allowed here so traversal can
// observe and report it; the approval validator rejects it independently.
// DECLARATION is rejected here too because internal analysis must not
// silently emit it as a source reference.
func validObservedReferenceClass(class ReferenceClass) bool {
	switch class {
	case refDirectCall, refFunctionValue, refMethodValue, refMethodExpression, refPackageVariable, refDotImport:
		return true
	default:
		return false
	}
}
