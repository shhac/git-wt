const std = @import("std");
const git = @import("git.zig");

pub const ValidationError = error{
    EmptyBranchName,
    BranchNameStartsWithDash,
    BranchNameHasSpaces,
    BranchNameHasInvalidChars,
    BranchNameTooLong,
    BranchNameReserved,
};

/// Validate a git branch name according to git naming rules
pub fn validateBranchName(branch_name: []const u8) ValidationError!void {
    if (branch_name.len == 0) {
        return ValidationError.EmptyBranchName;
    }
    
    if (branch_name.len > 250) {
        return ValidationError.BranchNameTooLong;
    }
    
    if (branch_name[0] == '-') {
        return ValidationError.BranchNameStartsWithDash;
    }
    
    // Check for spaces
    if (std.mem.indexOf(u8, branch_name, " ") != null) {
        return ValidationError.BranchNameHasSpaces;
    }
    
    // Check for invalid characters (as per git check-ref-format)
    // Also include shell metacharacters for additional safety
    const invalid_chars = "~^:?*[\\`;$&|()<>{}'\"";
    for (invalid_chars) |char| {
        if (std.mem.indexOf(u8, branch_name, &[_]u8{char}) != null) {
            return ValidationError.BranchNameHasInvalidChars;
        }
    }
    
    // Check for reserved names
    const reserved_names = [_][]const u8{
        "HEAD",
        "ORIG_HEAD",
        "FETCH_HEAD",
        "MERGE_HEAD",
        "refs",
        "remotes",
    };
    
    for (reserved_names) |reserved| {
        if (std.mem.eql(u8, branch_name, reserved)) {
            return ValidationError.BranchNameReserved;
        }
    }
    
    // Check for sequences that are problematic
    if (std.mem.indexOf(u8, branch_name, "..") != null) {
        return ValidationError.BranchNameHasInvalidChars;
    }
    
    if (std.mem.indexOf(u8, branch_name, "/.") != null) {
        return ValidationError.BranchNameHasInvalidChars;
    }
    
    if (std.mem.indexOf(u8, branch_name, "/.") != null) {
        return ValidationError.BranchNameHasInvalidChars;
    }
    
    if (std.mem.endsWith(u8, branch_name, "/")) {
        return ValidationError.BranchNameHasInvalidChars;
    }
    
    if (std.mem.startsWith(u8, branch_name, "/")) {
        return ValidationError.BranchNameHasInvalidChars;
    }
}

/// Get a human-readable error message for validation errors
pub fn getValidationErrorMessage(err: ValidationError) []const u8 {
    return switch (err) {
        ValidationError.EmptyBranchName => "Branch name cannot be empty",
        ValidationError.BranchNameStartsWithDash => "Branch name cannot start with '-'",
        ValidationError.BranchNameHasSpaces => "Branch name cannot contain spaces",
        ValidationError.BranchNameHasInvalidChars => "Branch name contains invalid characters (~^:?*[\\`;$&|()<>{}'\"..)",
        ValidationError.BranchNameTooLong => "Branch name is too long (max 250 characters)",
        ValidationError.BranchNameReserved => "Branch name is reserved (HEAD, ORIG_HEAD, etc.)",
    };
}

test "validateBranchName - valid names" {
    try validateBranchName("feature-branch");
    try validateBranchName("feature/new-ui");
    try validateBranchName("fix-123");
    try validateBranchName("v1.0.0");
}

test "validateBranchName - invalid names" {
    try std.testing.expectError(ValidationError.EmptyBranchName, validateBranchName(""));
    try std.testing.expectError(ValidationError.BranchNameStartsWithDash, validateBranchName("-feature"));
    try std.testing.expectError(ValidationError.BranchNameHasSpaces, validateBranchName("feature branch"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature~branch"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature..branch"));
    try std.testing.expectError(ValidationError.BranchNameReserved, validateBranchName("HEAD"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature/"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("/feature"));
    // Test shell metacharacters
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature;rm -rf"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature$USER"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature`date`"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature|cmd"));
    try std.testing.expectError(ValidationError.BranchNameHasInvalidChars, validateBranchName("feature&&cmd"));
}

test "getValidationErrorMessage" {
    const msg = getValidationErrorMessage(ValidationError.EmptyBranchName);
    try std.testing.expect(std.mem.eql(u8, msg, "Branch name cannot be empty"));
}

pub const ParentDirError = error{
    ParentDirNotFound,
    ParentDirNotDirectory,
    ParentDirNotWritable,
    ParentDirInsideRepo,
    PathTraversalAttempt,
    InvalidPath,
};

/// Validate parent directory for worktree creation
pub fn validateParentDir(allocator: std.mem.Allocator, path: []const u8, repo_info: git.RepoInfo) ![]u8 {
    // Basic path validation
    if (path.len == 0) {
        return ParentDirError.InvalidPath;
    }
    
    // Enhanced path traversal checks
    // Check for .. sequences
    if (std.mem.indexOf(u8, path, "..") != null) {
        return ParentDirError.PathTraversalAttempt;
    }
    
    // Check for URL-encoded traversal attempts
    const encoded_traversals = [_][]const u8{
        "%2e%2e", "%2e%2e/", "%2e%2e\\",
        "..%2f", "..%5c", "%252e%252e",
    };
    for (encoded_traversals) |pattern| {
        if (std.ascii.indexOfIgnoreCase(path, pattern) != null) {
            return ParentDirError.PathTraversalAttempt;
        }
    }
    
    // Check for null bytes which could truncate paths
    if (std.mem.indexOf(u8, path, "\x00") != null) {
        return ParentDirError.InvalidPath;
    }
    
    // Resolve to absolute path (handles symlinks and normalizes path)
    const abs_path = std.fs.realpathAlloc(allocator, path) catch |err| {
        switch (err) {
            error.FileNotFound => return ParentDirError.ParentDirNotFound,
            error.AccessDenied => return ParentDirError.ParentDirNotWritable,
            else => return err,
        }
    };
    errdefer allocator.free(abs_path);
    
    // Additional check: ensure resolved path doesn't contain ..
    if (std.mem.indexOf(u8, abs_path, "..") != null) {
        allocator.free(abs_path);
        return ParentDirError.PathTraversalAttempt;
    }
    
    // Ensure it's not inside the current repository
    if (std.mem.startsWith(u8, abs_path, repo_info.root)) {
        allocator.free(abs_path);
        return ParentDirError.ParentDirInsideRepo;
    }
    
    // Ensure it's not inside the main repository (if in worktree)
    if (repo_info.main_repo_root) |main_root| {
        if (std.mem.startsWith(u8, abs_path, main_root)) {
            allocator.free(abs_path);
            return ParentDirError.ParentDirInsideRepo;
        }
    }
    
    // Open directory to verify it exists and is a directory
    var dir = std.fs.openDirAbsolute(abs_path, .{}) catch |err| {
        switch (err) {
            error.NotDir => {
                allocator.free(abs_path);
                return ParentDirError.ParentDirNotDirectory;
            },
            error.AccessDenied => {
                allocator.free(abs_path);
                return ParentDirError.ParentDirNotWritable;
            },
            else => {
                allocator.free(abs_path);
                return err;
            },
        }
    };
    defer dir.close();
    
    // Test write permissions
    const test_name = try std.fmt.allocPrint(allocator, ".git-wt-test-{d}", .{std.time.milliTimestamp()});
    defer allocator.free(test_name);
    
    // Use a block to ensure proper cleanup
    const can_write = blk: {
        const test_file = dir.createFile(test_name, .{}) catch |err| {
            switch (err) {
                error.AccessDenied => break :blk false,
                else => {
                    allocator.free(abs_path);
                    return err;
                },
            }
        };
        test_file.close();
        dir.deleteFile(test_name) catch {}; // Best effort cleanup
        break :blk true;
    };
    
    if (!can_write) {
        allocator.free(abs_path);
        return ParentDirError.ParentDirNotWritable;
    }
    
    return abs_path; // Caller owns this memory
}

/// Get a human-readable error message for parent directory errors
pub fn getParentDirErrorMessage(err: ParentDirError) []const u8 {
    return switch (err) {
        ParentDirError.ParentDirNotFound => "Parent directory does not exist",
        ParentDirError.ParentDirNotDirectory => "Parent path is not a directory",
        ParentDirError.ParentDirNotWritable => "Parent directory is not writable",
        ParentDirError.ParentDirInsideRepo => "Parent directory cannot be inside the repository",
        ParentDirError.PathTraversalAttempt => "Path traversal attempts are not allowed",
        ParentDirError.InvalidPath => "Invalid parent directory path",
    };
}

test "validateParentDir" {
    const testing_allocator = std.testing.allocator;
    
    // Create a test directory
    var temp_dir = try std.fs.cwd().makeOpenPath("test-parent-dir", .{});
    defer temp_dir.close();
    defer std.fs.cwd().deleteDir("test-parent-dir") catch {};
    
    const repo_info = git.RepoInfo{
        .root = "/fake/repo",
        .name = "repo",
        .is_worktree = false,
        .main_repo_root = null,
    };
    
    // Test 1: Valid directory
    const result = try validateParentDir(testing_allocator, "test-parent-dir", repo_info);
    defer testing_allocator.free(result);
    try std.testing.expect(std.fs.path.isAbsolute(result));
    
    // Test 2: Non-existent directory
    const err = validateParentDir(testing_allocator, "non-existent-dir", repo_info);
    try std.testing.expectError(ParentDirError.ParentDirNotFound, err);
    
    // Test 3: Path traversal attempt
    const err2 = validateParentDir(testing_allocator, "../../../etc", repo_info);
    try std.testing.expectError(ParentDirError.PathTraversalAttempt, err2);
    
    // Test 4: Empty path
    const err3 = validateParentDir(testing_allocator, "", repo_info);
    try std.testing.expectError(ParentDirError.InvalidPath, err3);
    
    // Test 5: URL-encoded path traversal
    const err4 = validateParentDir(testing_allocator, "%2e%2e/etc", repo_info);
    try std.testing.expectError(ParentDirError.PathTraversalAttempt, err4);
    
    // Test 6: Mixed case URL encoding
    const err5 = validateParentDir(testing_allocator, "%2E%2e/etc", repo_info);
    try std.testing.expectError(ParentDirError.PathTraversalAttempt, err5);
    
    // Test 7: Null byte injection
    const err6 = validateParentDir(testing_allocator, "/tmp\x00/etc", repo_info);
    try std.testing.expectError(ParentDirError.InvalidPath, err6);
}