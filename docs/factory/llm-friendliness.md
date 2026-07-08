# LLM-Friendliness Gate

The LLM-friendliness verifier keeps the Leamas repository reviewable by LLMs, agents, and humans by rejecting oversized, dense, minified, generated-looking, or otherwise hard-to-review committed files.

## Why the Gate Exists

LLMs and agents are Factory participants, but they need clean, reviewable input to be effective. Large files, dense minified assets, and extremely long lines:

- Exhaust context windows quickly
- Make diffs and reviews hard to follow
- Hide meaningful changes in noise
- Break the agent-assisted development workflow

The LLM-friendliness gate enforces discipline that benefits both human reviewers and AI assistants.

## Default Thresholds

| Check | Threshold | Rationale |
|-------|-----------|-----------|
| File size | 64 KiB | Fits in context windows, keeps diffs small |
| Text lines | 400 | Single-purpose files stay focused |
| Line length | 240 chars | Readable without wrapping |
| Minified lines | 1000 chars | Flags dense CSS/JS/JSON for preprocessing |

## Structural Ignores

The verifier ignores these directories defensively:

- `.git/` - Git internals
- `build/` - Build artifacts
- `bin/` - Compiled binaries
- `vendor/` - External dependencies

These ignores are **structural only**. Files inside these directories are not checked.

## No Allowlist Rule

The verifier does **not** support:
- Per-file allowlists
- Exception lists
- Bypass globs
- Ignored large-file lists

If a file fails the gate:
1. Split it into smaller, focused files
2. Simplify or refactor the content
3. Remove generated output from the repository
4. Move binary assets to external storage if needed

This rule keeps the gate simple and prevents gradual erosion of standards.

## Minified File Detection

The gate flags long lines in files that are typically minified:

- `.js`, `.css`, `.html`, `.json`, `.xml`, `.svg`
- `.min.js`, `.min.css`

For these file types, any line over 1000 characters is flagged as minified-looking.

## Implementation: Go, Not Bash

The LLM-friendliness gate is implemented in Go (`internal/factory/llmfriendly/`), not Bash.

**Why Go:**
- Standard library has `filepath.WalkDir` and `bufio.Scanner`
- Git integration via `os/exec` is straightforward
- Fast, compiled, no runtime dependencies
- Fits Leamas Go-only doctrine

**Why not Bash:**
- Bash file processing is slow for large repos
- Line-by-line scanning requires careful quoting
- Parsing Git output in shell is error-prone
- The verifier is substantial automation, not "glue"

The tiny Bash wrapper (`scripts/verify_llm_friendliness.sh`) is under 50 LOC and delegates all logic to Go.

## Relation to Agent-Assisted Development

The LLM-friendliness gate directly supports agent-assisted development:

1. **Context efficiency**: Small files leave room for task context
2. **Diff clarity**: Short lines make changes visible
3. **Focus enforcement**: Line-count limits encourage single-purpose files
4. **No hidden content**: Binary/minified files don't contribute to understanding

Agents should also follow these principles when creating new files.

## How to Fix Violations

### Too Large (file > 64 KiB)
- Split the file into modules
- Extract configuration to external files
- Move test fixtures to `testdata/`
- Consider if the content belongs in the repo

### Too Many Lines (> 400 lines)
- Extract helper functions to separate files
- Split into logical modules
- Review if the file is doing too many things

### Long Lines (> 240 chars)
- Break long strings across lines
- Format JSON/XML with proper indentation
- Use shorter variable names if reasonable

### Minified Lines (> 1000 chars in minifiable files)
- Use source versions of libraries (unminified)
- Run a formatter before committing
- Preprocess assets in CI, not in the repo

## Verification Commands

```bash
# Run the verifier directly
./bin/leamas factory verify llm-friendly

# Or via the Make target
make verify-llm-friendly

# Via the wrapper script
chmod +x scripts/verify_llm_friendliness.sh
./scripts/verify_llm_friendliness.sh
```

## Exit Codes

- `0` - All files pass, LLM-friendliness verified
- `1` - One or more files fail, violations found

Output is deterministic and sorted by path for stable CI.

## References

- `internal/factory/llmfriendly/` - Go implementation
- `scripts/verify_llm_friendliness.sh` - Bash wrapper
- `docs/doctrine/agent-assisted-development.md` - Agent contract
- `docs/doctrine/go-only.md` - Go-only doctrine
- `docs/doctrine/factory-meta-loop.md` - Factory self-verification
