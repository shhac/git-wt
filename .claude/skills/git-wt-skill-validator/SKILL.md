---
name: git-wt-skill-validator
description: This skill should be used to validate that project skills accurately reflect the current codebase state and update them when drift is detected. Invoked when checking skill accuracy, detecting documentation drift, updating skills after refactoring, or ensuring skills match current architecture. Use proactively after significant code changes.
allowed-tools: Read, Grep, Glob, Edit, Write, Bash
---

# git-wt Skill Validator

Validates and updates project skills to ensure they accurately reflect the current codebase state.

## When to Use This Skill

**Automatic Triggers:**
- After significant architecture changes (new modules, renamed files, restructured directories)
- After refactoring that affects project structure or workflows
- When adding/removing commands or major features
- After release process changes
- Periodically (monthly or quarterly) to catch drift

**User Requests:**
- "Validate our skills are up to date"
- "Check if skills match current codebase"
- "Update skill documentation"
- "Skills seem out of sync"

## Validation Process

### 1. Identify Skills to Validate

Available skills in `.claude/skills/`:
- `git-wt-architecture` - Codebase structure, file locations, design patterns
- `git-wt-bugtrack` - Bug tracking workflow, BUGS.md structure
- `git-wt-debug` - Debugging procedures, troubleshooting guide
- `git-wt-release` - Release process, versioning, CI/CD
- `git-wt-test` - Testing workflows, test commands, coverage

### 2. Validation Checks by Skill

#### git-wt-architecture
**Check:**
- Directory structure in `src/` matches documented structure
- Command files in `src/commands/` match listed commands
- Utility modules in `src/utils/` match documented count (14 modules)
- Design patterns mentioned still apply
- Entry points and dispatch mechanisms accurate

**Commands:**
```bash
# List actual command files
ls -1 src/commands/*.zig | wc -l
ls -1 src/commands/*.zig

# List actual utility modules
ls -1 src/utils/*.zig | wc -l
ls -1 src/utils/*.zig

# Check for new directories
find src -type d -maxdepth 2
```

**Key Sections to Verify:**
- Project structure diagram
- Command list (new, remove, go, list, alias, clean, setup)
- Utility module count and names
- Design patterns (command table, GitResult union, etc.)

#### git-wt-release
**Check:**
- Version in `build.zig` matches current version
- Release workflow steps match `.github/workflows/release.yml`
- CHANGELOG.md structure matches documented format
- Dependencies list is current
- Build commands are correct

**Commands:**
```bash
# Check current version
grep 'version_option.*orelse' build.zig

# Verify GitHub Actions workflows exist
ls -1 .github/workflows/

# Check CHANGELOG structure
head -20 CHANGELOG.md
```

**Key Sections to Verify:**
- Version number (should match latest in CHANGELOG.md)
- 6-step release process
- GitHub Actions workflow names
- Required files (CHANGELOG.md, build.zig, etc.)

#### git-wt-test
**Check:**
- Test command accuracy (`zig test src/main.zig`, etc.)
- Test coverage numbers match actual test count
- Test file locations accurate
- Integration test count matches `src/integration_tests.zig`

**Commands:**
```bash
# Count unit tests (look for "test " declarations)
rg "^test \"" src/main.zig src/commands/*.zig src/utils/*.zig | wc -l

# Count integration tests
rg "^test \"" src/integration_tests.zig | wc -l

# Verify test files exist
ls -1 src/*_test.zig 2>/dev/null || echo "No standalone test files"
```

**Key Sections to Verify:**
- Test coverage numbers (70 unit tests, 38 integration tests)
- Test command syntax
- Test file locations
- Expect-based test references

#### git-wt-debug
**Check:**
- Debug flags match current `--debug` implementation
- Common issues still relevant
- Troubleshooting steps reference existing files/scripts
- Debug output examples match current format

**Commands:**
```bash
# Check available flags
./zig-out/bin/git-wt --help | grep -A 20 "Global Flags"

# Verify debug scripts exist
ls -1 debugging/*.sh 2>/dev/null || echo "No debug scripts"

# Test debug output
./zig-out/bin/git-wt --debug --version
```

**Key Sections to Verify:**
- Global flags (--debug, --no-tty, --no-color, --plain)
- Debug script references
- Common issues list

#### git-wt-bugtrack
**Check:**
- BUGS.md structure matches documented template
- Bug lifecycle states accurate
- Current bug count matches skill claim
- Bug tracking workflow still applies

**Commands:**
```bash
# Check BUGS.md structure
head -30 BUGS.md

# Count open bugs
rg "^## Bug" BUGS.md | wc -l

# Count fixed bugs
rg "Status: Fixed" BUGS.md | wc -l
```

**Key Sections to Verify:**
- BUGS.md template structure
- Bug status values (Open, In Progress, Fixed, Won't Fix)
- Documentation workflow
- Example bug entries

### 3. Detect Drift

For each validation check, compare:
- **Expected** (documented in skill)
- **Actual** (current codebase state)

**Drift Categories:**
1. **Minor Drift** - Numbers changed (test count, module count, bug count)
2. **Moderate Drift** - New features added, commands renamed
3. **Major Drift** - Architecture changed, workflows revised, design patterns altered

### 4. Report Findings

Present findings in this format:

```markdown
## Skill Validation Report

### git-wt-architecture
**Status:** [✅ Accurate | ⚠️ Minor Drift | ❌ Major Drift]

**Findings:**
- Directory structure: ✅ Matches
- Command count: ⚠️ Documented: 6, Actual: 7 (added 'setup')
- Utility modules: ✅ 14 modules confirmed

**Recommended Actions:**
- Update command list to include 'setup' command
- Add 'setup' to command table example

### git-wt-test
**Status:** ⚠️ Minor Drift

**Findings:**
- Unit test count: ❌ Documented: 70, Actual: 73
- Integration test count: ✅ 38 confirmed

**Recommended Actions:**
- Update test coverage numbers
```

### 5. Update Skills (Optional)

**When to Auto-Update:**
- Minor drift (numbers, counts, versions)
- User explicitly requests updates
- Drift is unambiguous (clear what changed)

**When to Ask User:**
- Major architectural changes (need design decision)
- Moderate drift that might be intentional
- Multiple valid interpretations of changes
- User preference for manual review

**Update Process:**
```bash
# 1. Read current skill
# 2. Apply changes using Edit tool
# 3. Verify changes with diff
# 4. Commit if requested
```

## Validation Modes

### Quick Validation
- Check only critical facts (counts, versions, file existence)
- Report pass/fail per skill
- ~30 seconds

### Deep Validation
- Read skill content in detail
- Compare each section against codebase
- Generate detailed drift report
- ~2-3 minutes

### Auto-Update Mode
- Run deep validation
- Automatically fix minor drift
- Ask about moderate/major drift
- Commit changes after user approval
- ~5 minutes

## Example Usage

### User Request: "Validate our skills"
```
1. Run quick validation on all skills
2. Report any drift detected
3. Ask if user wants detailed report or auto-update
```

### User Request: "Update skills after refactor"
```
1. Run deep validation
2. Identify all drift (likely major)
3. Present findings with recommended changes
4. Ask for approval to update
5. Apply updates and commit
```

### Proactive (After Major Commit)
```
Hook trigger: After commit with "refactor" or "feat" in message
1. Run quick validation
2. If drift detected, notify user
3. Offer to run deep validation
```

## Implementation Notes

### Validation Order
1. git-wt-architecture (foundation - validate first)
2. git-wt-test (depends on architecture)
3. git-wt-release (depends on build system)
4. git-wt-debug (depends on flags/features)
5. git-wt-bugtrack (independent)

### Handling Edge Cases
- **Skill doesn't exist**: Skip with warning
- **Codebase file missing**: Report as major drift
- **Ambiguous counts**: Use heuristics (rg patterns), report uncertainty
- **Build required**: Run `zig build` first if binary doesn't exist

### Best Practices
- Always build project first to ensure binary is current
- Use exact grep/rg patterns to avoid false positives
- Compare structural elements, not prose
- Focus on facts (counts, file paths, command names), not descriptions
- When uncertain, report finding rather than auto-updating

## Success Criteria

**Validation is successful when:**
- All skills checked against current codebase
- Drift identified with specific examples
- User informed of status
- Updates applied if requested and appropriate

**Validation report should include:**
- Skill name and status (✅/⚠️/❌)
- Specific findings (what changed)
- Recommended actions (what to update)
- Confidence level (high/medium/low)

## Related Skills

- **git-wt-architecture** - Primary target for validation
- **git-wt-test** - Test counts need frequent updates
- **git-wt-release** - Version numbers change with releases
- All skills - Can be validated and updated by this skill
