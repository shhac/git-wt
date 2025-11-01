---
name: git-wt-bugtrack
description: This skill should be used when documenting bugs, tracking issues, updating BUGS.md, or reviewing known problems in git-wt. Invoked when adding bugs to BUGS.md, documenting issues, tracking edge cases, or reviewing bug status.
allowed-tools: Read, Edit, Grep
---

# git-wt Bug Tracking Workflow

Systematically document, track, and resolve bugs in the git-wt codebase using BUGS.md.

## Instructions

### 1. Understanding BUGS.md Structure

The BUGS.md file tracks all identified bugs using a numbered system and categories:

**Categories:**
- **Fixed Issues** - Resolved bugs (kept for historical reference)
- **Critical Issues** - Severe bugs affecting core functionality
- **Edge Cases** - Unusual scenarios that may fail
- **Usability Issues** - UX problems that need improvement
- **Code Quality Issues** - Technical debt or refactoring needs
- **Performance Issues** - Slow operations or inefficiencies
- **Platform-Specific Issues** - OS or environment-specific problems
- **Documentation Issues** - Missing or incorrect documentation
- **Testing Gaps** - Areas lacking test coverage
- **Security Issues** - Security vulnerabilities or concerns

### 2. Adding a New Bug

When discovering a bug:

**Step 1: Determine severity and category**
- Is it critical? (affects core features)
- Is it an edge case? (rare scenario)
- Is it platform-specific? (macOS/Linux/Windows)

**Step 2: Assign bug number**
- Check highest bug number in Fixed Issues
- Use next sequential number

**Step 3: Document the bug**

```markdown
## [Category Name]

### Bug #XX: Brief Description

**Issue:** Clear description of the problem

**Impact:** How this affects users or system
- Severity level (Critical/High/Medium/Low)
- Affected features
- Frequency of occurrence

**Reproduction:**
1. Step-by-step to reproduce
2. Expected behavior
3. Actual behavior

**Example:**
\```bash
# Commands that trigger the bug
git-wt command --flag
# Output or error
\```

**Suggested Fix:** Proposed solution or approach
- Technical approach
- Files affected
- Potential side effects
```

### 3. Bug Number Reference

**Historical bugs (all fixed):**
- Bugs #1-#46 are documented in Fixed Issues section
- Reference these when encountering similar issues
- Learn from past solutions

**Next bug number:** Start from #47

### 4. Researching Existing Bugs

Before adding a bug, check if it's already documented:

```bash
# Search for similar bugs
grep -i "keyword" BUGS.md

# Check fixed issues
grep -A10 "Bug #XX" BUGS.md

# Search by category
grep -A5 "## Critical Issues" BUGS.md
```

### 5. Updating Bug Status

**When bug is fixed:**

1. Move from active category to Fixed Issues
2. Update with ✅ symbol
3. Add resolution details
4. Reference commit that fixed it

```markdown
## Fixed Issues
- ✅ Bug #XX: Description (resolution details)
```

**When bug severity changes:**
- Move between categories
- Update description with new information

### 6. Bug Lifecycle

```
Discovery → Documentation → Categorization → Fix → Verification → Archive
```

1. **Discovery:** Bug found during development or testing
2. **Documentation:** Added to BUGS.md with full details
3. **Categorization:** Placed in appropriate section
4. **Fix:** Code changes made, tests added
5. **Verification:** Bug fix tested and confirmed
6. **Archive:** Moved to Fixed Issues with ✅

### 7. Reviewing Code for Bugs

Systematic code review to find bugs:

```bash
# Search for common bug patterns
grep -r "TODO\|FIXME\|XXX" src/

# Look for error handling
grep -r "catch\|error\|panic" src/

# Check for memory management
grep -r "allocator\|free\|defer" src/

# Find potential race conditions
grep -r "lock\|mutex\|atomic" src/
```

### 8. Cross-Referencing DESIGN.md

When fixing bugs, ensure solutions conform to design principles:
- Zero runtime dependencies
- Clear, maintainable code
- Proper error handling
- Cross-platform compatibility

Read DESIGN.md before proposing fixes.

## Bug Documentation Template

```markdown
### Bug #XX: [Brief Title]

**Issue:**
[Clear description of the problem]

**Impact:**
- Severity: [Critical/High/Medium/Low]
- Affects: [Feature/Component affected]
- Frequency: [Always/Often/Rare]

**Reproduction:**
1. [Step 1]
2. [Step 2]
3. [Observe: Expected vs Actual]

**Example:**
\```bash
# Minimal reproduction
git-wt command args
# Error output
\```

**Suggested Fix:**
- [Technical approach]
- [Files to modify]
- [Potential risks]

**Related:**
- Similar to Bug #YY
- Affects same code as Bug #ZZ
```

## Categories in Detail

### Critical Issues
- Crashes or data loss
- Core features completely broken
- Security vulnerabilities
- Fix immediately

### Edge Cases
- Unusual but valid scenarios
- Rare combinations of inputs
- Boundary conditions
- Document workaround if can't fix immediately

### Usability Issues
- Confusing error messages
- Poor user experience
- Missing feedback
- Can work but needs improvement

### Code Quality Issues
- Technical debt
- Duplicate code
- Complex functions needing refactoring
- Not urgent but improves maintainability

### Performance Issues
- Slow operations
- Memory leaks
- Unnecessary computation
- Affects large repositories

### Platform-Specific Issues
- macOS-only or Linux-only
- Terminal compatibility
- Path handling differences
- Test on target platform

### Documentation Issues
- Missing docs
- Incorrect information
- Outdated examples
- Update docs when fixing

### Testing Gaps
- Features without tests
- Edge cases not tested
- Integration scenarios missing
- Add tests when fixing

### Security Issues
- Injection vulnerabilities
- Path traversal
- Privilege escalation
- Fix immediately, consider security advisory

## Common Bug Patterns

### Memory Leaks
```zig
// Bug: Missing free
const output = try git.exec(allocator, args);
// ... use output ...
// BUG: Never freed!

// Fix: Add defer
const output = try git.exec(allocator, args);
defer allocator.free(output);
```

### Error Handling
```zig
// Bug: Swallowed error
doOperation() catch {};

// Fix: Proper handling
doOperation() catch |err| {
    // Log or return error
    return err;
};
```

### Race Conditions
```zig
// Bug: TOCTOU (Time-of-check to time-of-use)
if (fileExists(path)) {
    // File might be deleted here!
    readFile(path);
}

// Fix: Handle error
readFile(path) catch |err| {
    // Handle file not found
};
```

## Bug Triage

**When reviewing bugs:**

1. **Verify reproducibility**
   - Can you reproduce it?
   - Under what conditions?

2. **Assess impact**
   - How many users affected?
   - Workaround available?

3. **Determine priority**
   - Critical → Fix immediately
   - High → Fix in next release
   - Medium → Fix when possible
   - Low → Backlog

4. **Assign to release**
   - Add to TODO.md if planned
   - Document workaround if deferred

## Closing Bugs

When all bugs in a category are fixed:

```markdown
## [Category Name]

(None currently identified)
```

This shows active maintenance and clean slate.

## References

- **BUGS.md** - Main bug tracking file
- **TODO.md** - Planned bug fixes
- **CHANGELOG.md** - Bug fixes per release
- **DESIGN.md** - Design principles for solutions
- **GitHub Issues** - External bug reports
