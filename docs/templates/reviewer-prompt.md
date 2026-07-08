# Reviewer Prompt Template

> Guidance for reviewers on what to focus on.

---

## For Code Review

### Focus Areas

1. **Correctness**
   - Does the code do what the ACT specifies?
   - Are edge cases handled?

2. **Safety**
   - Are there security concerns?
   - Could this break existing functionality?

3. **Style**
   - Does it follow existing patterns?
   - Is it readable?

### Red Flags

- ❌ Missing error handling
- ❌ Hardcoded values that should be configurable
- ❌ Duplicate code that could be refactored
- ❌ Missing tests for critical paths

### Green Flags

- ✅ Clear function/variable names
- ✅ Appropriate comments for complex logic
- ✅ Tests for new functionality
- ✅ Updates to documentation

---

## For Documentation Review

### Focus Areas

1. **Clarity**
   - Can a new developer understand this?
   - Are there ambiguous terms?

2. **Completeness**
   - Are all steps covered?
   - Are edge cases documented?

3. **Accuracy**
   - Does this match the implementation?
   - Are links valid?

### Questions to Ask

- Would I know how to use this if I came back in 6 months?
- Is the "why" explained, not just the "what"?
- Are there examples where helpful?
