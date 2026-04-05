package main

import (
	"strings"
	"testing"
)

func TestSanitizeSearchString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "long string",
			input:    strings.Repeat("a", 1500),
			expected: strings.Repeat("a", 1000),
		},
		{
			name:     "dangerous patterns",
			input:    "hello<script>alert('xss')</script>world",
			expected: "helloalert(xss)world",
		},
		{
			name:     "path traversal",
			input:    "../../../etc/passwd",
			expected: "etc/passwd",
		},
		{
			name:     "whitespace trimming",
			input:    "  hello world  ",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeSearchString(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeSearchString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "empty path",
			input:       "",
			expected:    ".",
			expectError: false,
		},
		{
			name:        "root path",
			input:       "/",
			expected:    ".",
			expectError: false,
		},
		{
			name:        "normal path",
			input:       "/blog/test.md",
			expected:    "blog/test.md",
			expectError: false,
		},
		{
			name:        "path traversal attempt",
			input:       "/../../../etc/passwd",
			expected:    "",
			expectError: true,
		},
		{
			name:        "git directory",
			input:       "/.git/config",
			expected:    "",
			expectError: true,
		},
		{
			name:        "relative path with dot",
			input:       "/./blog/test.md",
			expected:    "blog/test.md",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("validatePath(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("validatePath(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("validatePath(%q) = %q, want %q", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestGetContentType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "html",
			input:    "html",
			expected: "text/html;charset=utf-8",
		},
		{
			name:     "css",
			input:    "css",
			expected: "text/css",
		},
		{
			name:     "js",
			input:    "js",
			expected: "application/x-javascript",
		},
		{
			name:     "pdf",
			input:    "pdf",
			expected: "application/pdf",
		},
		{
			name:     "unknown",
			input:    "unknown",
			expected: "text/html;charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getContentType(tt.input)
			if result != tt.expected {
				t.Errorf("getContentType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		sep      string
		expected []string
	}{
		{
			name:     "normal split",
			s:        "a.b.c",
			sep:      ".",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty strings",
			s:        "...",
			sep:      ".",
			expected: []string{},
		},
		{
			name:     "mixed",
			s:        "a..b.c",
			sep:      ".",
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := split(tt.s, tt.sep)
			if len(result) != len(tt.expected) {
				t.Errorf("split(%q, %q) length = %d, want %d", tt.s, tt.sep, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("split(%q, %q)[%d] = %q, want %q", tt.s, tt.sep, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestIsDir(t *testing.T) {
	// 测试当前目录
	if !isDir(".") {
		t.Error("isDir(\".\") should return true")
	}

	// 测试不存在的目录
	if isDir("nonexistent_directory_12345") {
		t.Error("isDir(\"nonexistent_directory_12345\") should return false")
	}
}

func TestIsFile(t *testing.T) {
	// 测试当前文件（main.go应该存在）
	if !isFile("main.go") {
		t.Error("isFile(\"main.go\") should return true")
	}

	// 测试目录
	if isFile(".") {
		t.Error("isFile(\".\") should return false")
	}
}
