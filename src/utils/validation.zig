const std = @import("std");

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
    const invalid_chars = "~^:?*[\\";
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
        ValidationError.BranchNameHasInvalidChars => "Branch name contains invalid characters (~^:?*[\\..)",
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
}

test "getValidationErrorMessage" {
    const msg = getValidationErrorMessage(ValidationError.EmptyBranchName);
    try std.testing.expect(std.mem.eql(u8, msg, "Branch name cannot be empty"));
}