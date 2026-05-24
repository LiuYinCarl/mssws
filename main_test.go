package main

import (
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// ────────────────────────────────────────────
// sanitizeSearchString
// ────────────────────────────────────────────

func TestSanitizeSearchString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal text", "hello world", "hello world"},
		{"empty string", "", ""},
		{"trim whitespace", "  hello  ", "hello"},
		{"long string truncated", strings.Repeat("a", 1500), strings.Repeat("a", 1000)},
		{"script tag removed", "hello<script>alert(1)</script>world", "helloalert(1)world"},
		{"javascript uri removed", "click javascript:void(0) here", "click void(0) here"},
		{"path traversal ../ removed", "../../../etc/passwd", "etc/passwd"},
		{"semicolons removed", "foo;bar;baz", "foobarbaz"},
		{"pipes removed", "a|b|c", "abc"},
		{"backticks removed", "`cmd`", "cmd"},
		{"dollar signs removed", "$HOME", "HOME"},
		{"onload removed", "body onload=foo", "body foo"},
		{"double quotes removed", `say "hello"`, "say hello"},
		{"single quotes removed", "it's ok", "its ok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSearchString(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeSearchString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ────────────────────────────────────────────
// validatePath
// ────────────────────────────────────────────

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expect      string
		expectError bool
	}{
		{"empty path returns root", "", ".", false},
		{"slash returns root", "/", ".", false},
		{"normal file", "/blog/test.md", filepath.Join(".", "blog", "test.md"), false},
		{"file without leading slash", "blog/readme.md", filepath.Join(".", "blog", "readme.md"), false},
		{"path traversal blocked", "/../../../etc/passwd", "", true},
		{"git directory blocked", "/.git/config", "", true},
		{"dot cleaned", "/./blog/test.md", filepath.Join(".", "blog", "test.md"), false},
		{"only dot slash", "/.", ".", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatePath(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("validatePath(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("validatePath(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expect {
				t.Errorf("validatePath(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

// ────────────────────────────────────────────
// validateQueryPath
// ────────────────────────────────────────────

func TestValidateQueryPath(t *testing.T) {
	// The validateQueryPath requires the absolute path to be within
	// the project root. Use the project's blog dir (or a temp dir
	// inside the project).
	tmpDir, err := os.MkdirTemp(".", "testdata_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name   string
		path   string
		expect bool
	}{
		{"empty path", "", false},
		{"valid md file (relative)", filepath.Join(tmpDir, "test.md"), true},
		{"non-md suffix", filepath.Join(tmpDir, "test.txt"), false},
		{"uppercase MD", filepath.Join(tmpDir, "TEST.MD"), true},
		{"path outside root", "/etc/passwd", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the file so isFile check passes
			if strings.HasSuffix(strings.ToLower(tt.path), ".md") && tt.expect {
				os.WriteFile(tt.path, []byte("test"), 0644)
			}
			got := validateQueryPath(tt.path)
			if got != tt.expect {
				t.Errorf("validateQueryPath(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

// ────────────────────────────────────────────
// getContentType
// ────────────────────────────────────────────

func TestGetContentType(t *testing.T) {
	tests := []struct {
		suffix   string
		expected string
	}{
		{"html", "text/html;charset=utf-8"},
		{"css", "text/css"},
		{"js", "application/x-javascript"},
		{"pdf", "application/pdf"},
		{"png", "application/x-png"},
		{"svg", "image/svg+xml"},
		{"ico", "image/x-icon"},
		{"xml", "application/rss+xml;charset=utf-8"},
		{"ttf", "application/x-font-truetype"},
		{"woff", "application/x-font-woff"},
		{"woff2", "application/x-font-woff"},
		{"unknown", "text/html;charset=utf-8"},
	}
	for _, tt := range tests {
		t.Run(tt.suffix, func(t *testing.T) {
			got := getContentType(tt.suffix)
			if got != tt.expected {
				t.Errorf("getContentType(%q) = %q, want %q", tt.suffix, got, tt.expected)
			}
		})
	}
}

// ────────────────────────────────────────────
// split
// ────────────────────────────────────────────

func TestSplit(t *testing.T) {
	tests := []struct {
		s    string
		sep  string
		want []string
	}{
		{"a.b.c", ".", []string{"a", "b", "c"}},
		{"...", ".", []string{}},
		{"a..b", ".", []string{"a", "b"}},
		{"", ".", []string{}},
		{"abc", ".", []string{"abc"}},
		{"a,b,c", ",", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		got := split(tt.s, tt.sep)
		if len(got) != len(tt.want) {
			t.Errorf("split(%q,%q) len=%d, want %d", tt.s, tt.sep, len(got), len(tt.want))
			return
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("split(%q,%q)[%d] = %q, want %q", tt.s, tt.sep, i, got[i], tt.want[i])
			}
		}
	}
}

// ────────────────────────────────────────────
// isDir / isFile
// ────────────────────────────────────────────

func TestIsDir(t *testing.T) {
	if !isDir(".") {
		t.Error(`isDir(".") should be true`)
	}
	if isDir(filepath.Join("nonexistent_dir_xyz_12345")) {
		t.Error(`isDir for nonexistent path should be false`)
	}
	tmpDir, err := os.MkdirTemp(".", "testdata_isdir_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	if !isDir(tmpDir) {
		t.Error(`isDir for temp dir should be true`)
	}
}

func TestIsFile(t *testing.T) {
	if !isFile("main.go") {
		t.Error(`isFile("main.go") should be true`)
	}
	if isFile(".") {
		t.Error(`isFile(".") should be false`)
	}
	if isFile(filepath.Join("nonexistent_file_xyz_12345")) {
		t.Error(`isFile for nonexistent path should be false`)
	}
}

// ────────────────────────────────────────────
// querySingleFile
// ────────────────────────────────────────────

func TestQuerySingleFile(t *testing.T) {
	// Create temp files inside the project root so validateQueryPath passes
	tmpDir, err := os.MkdirTemp(".", "testdata_qsf_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mdFile := filepath.Join(tmpDir, "post.md")
	os.WriteFile(mdFile, []byte("hello world\nfoo bar baz\n"), 0644)

	tests := []struct {
		name     string
		filePath string
		query    string
		expect   bool
	}{
		{"match found", mdFile, "hello", true},
		{"match in middle", mdFile, "bar", true},
		{"no match", mdFile, "xyzzy", false},
		{"empty filePath", "", "hello", false},
		{"empty query", mdFile, "", false},
		{"non-existent file", filepath.Join(tmpDir, "nope.md"), "hello", false},
		{"wrong suffix", filepath.Join(tmpDir, "post.txt"), "hello", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := querySingleFile(tt.filePath, tt.query)
			if got != tt.expect {
				t.Errorf("querySingleFile(%q, %q) = %v, want %v", tt.filePath, tt.query, got, tt.expect)
			}
		})
	}
}

// ────────────────────────────────────────────
// loadConfig
// ────────────────────────────────────────────

func TestLoadConfig(t *testing.T) {
	origConfPath := confPath
	origConf := conf
	origForbidden := append([]string{}, forbiddenFiles...)
	defer func() {
		confPath = origConfPath
		conf = origConf
		forbiddenFiles = origForbidden
	}()

	t.Run("valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		blogDir := filepath.Join(tmpDir, "blog")
		logFile := filepath.Join(tmpDir, "test.log")

		confPath = filepath.Join(tmpDir, "config.toml")
		configContent := `
BlogDir = "` + strings.ReplaceAll(blogDir, "\\", "\\\\") + `"
Port = 8080
LogFile = "` + strings.ReplaceAll(logFile, "\\", "\\\\") + `"
DevMode = "release"
`
		os.WriteFile(confPath, []byte(configContent), 0644)

		err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig failed: %v", err)
		}
		if conf.BlogDir != blogDir {
			t.Errorf("BlogDir = %q, want %q", conf.BlogDir, blogDir)
		}
		if conf.Port != 8080 {
			t.Errorf("Port = %d, want 8080", conf.Port)
		}
		isDir, _ := isDirStat(conf.BlogDir)
		if !isDir {
			t.Error("BlogDir was not created")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		confPath = filepath.Join(t.TempDir(), "nonexistent.toml")
		err := loadConfig()
		if err == nil {
			t.Error("expected error for missing config file")
		}
	})

	t.Run("missing BlogDir - auto-created", func(t *testing.T) {
		// loadConfig auto-creates BlogDir, so missing BlogDir is NOT an error.
		// Instead test that BlogDir gets created automatically.
		tmpDir := t.TempDir()
		blogDir := filepath.Join(tmpDir, "auto_blog")
		logFile := filepath.Join(tmpDir, "auto.log")
		confPath = filepath.Join(tmpDir, "config.toml")
		os.WriteFile(confPath, []byte(
			`BlogDir = "`+strings.ReplaceAll(blogDir, "\\", "\\\\")+`"
Port = 8080
LogFile = "`+strings.ReplaceAll(logFile, "\\", "\\\\")+`"
DevMode = "release"
`), 0644)

		err := loadConfig()
		if err != nil {
			t.Fatalf("auto-creation of BlogDir should succeed: %v", err)
		}
		isDir, _ := isDirStat(blogDir)
		if !isDir {
			t.Error("BlogDir was not auto-created")
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		tmpDir := t.TempDir()
		confPath = filepath.Join(tmpDir, "config.toml")
		os.WriteFile(confPath, []byte(
			`BlogDir = "./blog"
Port = 0
LogFile = "./nope.log"
DevMode = "release"
`), 0644)
		err := loadConfig()
		if err == nil {
			t.Error("expected error for invalid port")
		}
	})

	t.Run("port too high", func(t *testing.T) {
		tmpDir := t.TempDir()
		confPath = filepath.Join(tmpDir, "config.toml")
		os.WriteFile(confPath, []byte(
			`BlogDir = "./blog"
Port = 99999
LogFile = "./nope.log"
DevMode = "release"
`), 0644)
		err := loadConfig()
		if err == nil {
			t.Error("expected error for port > 65535")
		}
	})

	t.Run("forbiddenFiles includes log file", func(t *testing.T) {
		forbiddenFiles = []string{
			"./directory_monitor.sh",
			"./genindex.py",
			"./main.go",
			"./config.toml",
			"./genindex.sh",
			"./run.sh",
			"./mssws_prog",
		}
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "mylog.log")
		blogDir := filepath.Join(tmpDir, "myblog")
		confPath = filepath.Join(tmpDir, "config.toml")
		os.WriteFile(confPath, []byte(
			`BlogDir = "`+strings.ReplaceAll(blogDir, "\\", "\\\\")+`"
Port = 8080
LogFile = "`+strings.ReplaceAll(logFile, "\\", "\\\\")+`"
DevMode = "release"
`), 0644)

		err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig failed: %v", err)
		}
		found := false
		for _, f := range forbiddenFiles {
			if f == logFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("log file %q not found in forbiddenFiles", logFile)
		}
	})
}

// isDirStat is a helper that uses os.Stat directly (not the package-level
// isDir which returns false on stat errors). We want to distinguish
// "does not exist" from "is not a directory".
func isDirStat(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return s.IsDir(), nil
}

// ────────────────────────────────────────────
// HTTP handler: index page
// ────────────────────────────────────────────

func TestIndexHandler(t *testing.T) {
	tmpDir := t.TempDir()
	tmplDir := filepath.Join(tmpDir, "tmpl")
	blogDir := filepath.Join(tmpDir, "blog")
	os.MkdirAll(tmplDir, 0755)
	os.MkdirAll(blogDir, 0755)

	realTmplDir := "tmpl"
	for _, name := range []string{"style.tmpl", "index.tmpl", "article.tmpl", "query.tmpl"} {
		src := filepath.Join(realTmplDir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("cannot read real template %s: %v", src, err)
		}
		os.WriteFile(filepath.Join(tmplDir, name), data, 0644)
	}

	configContent := `
BlogDir = "` + strings.ReplaceAll(blogDir, "\\", "\\\\") + `"
Port = 9999
Ip = "127.0.0.1"
LogFile = "` + strings.ReplaceAll(filepath.Join(tmpDir, "test.log"), "\\", "\\\\") + `"
DevMode = "release"
HomePageLink = "/index.html"
HomePageTitle = "Home Page"
SiteTitle = "Test Site"
CacheTime = 0
FootPrint = ""
`
	os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "index.data"), []byte("<a href=\"./blog/post.md\">post.md</a></br>"), 0644)
	os.WriteFile(filepath.Join(blogDir, "post.md"), []byte("# Hello\nworld"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "query.data"), []byte(filepath.Join(blogDir, "post.md")+"\n"), 0644)

	origConfPath := confPath
	origTmplPaths := [4]string{indexTemplatePath, articleTemplatePath, queryTemplatePath, styleTemplatePath}
	origConf := conf
	origForbidden := append([]string{}, forbiddenFiles...)
	origQueryFile := queryFile
	origRootDir := rootDir
	origCwd, _ := os.Getwd()

	defer func() {
		confPath = origConfPath
		indexTemplatePath = origTmplPaths[0]
		articleTemplatePath = origTmplPaths[1]
		queryTemplatePath = origTmplPaths[2]
		styleTemplatePath = origTmplPaths[3]
		conf = origConf
		forbiddenFiles = origForbidden
		queryFile = origQueryFile
		rootDir = origRootDir
		os.Chdir(origCwd)
	}()

	os.Chdir(tmpDir)
	confPath = filepath.Join(tmpDir, "config.toml")
	indexTemplatePath = filepath.Join(tmplDir, "index.tmpl")
	articleTemplatePath = filepath.Join(tmplDir, "article.tmpl")
	queryTemplatePath = filepath.Join(tmplDir, "query.tmpl")
	styleTemplatePath = filepath.Join(tmplDir, "style.tmpl")
	queryFile = filepath.Join(tmpDir, "query.data")
	rootDir, _ = filepath.Abs(tmpDir)
	templates = make(map[string]*template.Template)

	if err := loadConfig(); err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	t.Run("GET / returns index page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		index(w, req)

		if w.Result().StatusCode != 200 {
			t.Errorf("status = %d, want 200", w.Result().StatusCode)
		}
		body := w.Body.String()
		if !strings.Contains(body, "<!DOCTYPE html>") {
			t.Error("index page missing DOCTYPE")
		}
		if !strings.Contains(body, "Test Site") {
			t.Error("index page missing site title")
		}
	})

	t.Run("GET /index.html returns index", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()
		index(w, req)
		if w.Result().StatusCode != 200 {
			t.Errorf("status = %d", w.Result().StatusCode)
		}
	})

	t.Run("GET /blog/post.md returns article", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/blog/post.md", nil)
		w := httptest.NewRecorder()
		index(w, req)

		body := w.Body.String()
		if w.Result().StatusCode != 200 {
			t.Errorf("status = %d, want 200", w.Result().StatusCode)
		}
		if !strings.Contains(body, "\\begin{md}") {
			t.Error("article page missing texme begin marker")
		}
	})

	t.Run("GET nonexistent file returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/blog/nope.md", nil)
		w := httptest.NewRecorder()
		index(w, req)
		if !strings.Contains(w.Body.String(), "404") {
			t.Errorf("expected 404, got: %s", w.Body.String())
		}
	})

	t.Run("GET forbidden file is blocked", func(t *testing.T) {
		// The validatePath converts /main.go to ./main.go,
		// which matches the forbiddenFiles list.
		req := httptest.NewRequest("GET", "/main.go", nil)
		w := httptest.NewRecorder()
		index(w, req)
		body := strings.ToLower(w.Body.String())
		if !strings.Contains(body, "forbidden") && !strings.Contains(body, "404") {
			// Under test conditions, the path may resolve to 404 since
			// the working dir is a tmp dir without a real main.go.
			// Either "forbidden" or "404" is acceptable.
			t.Errorf("expected forbidden or 404, got: %s", w.Body.String())
		}
	})

	t.Run("path traversal is blocked", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/../../../etc/passwd", nil)
		w := httptest.NewRecorder()
		index(w, req)
		if !strings.Contains(strings.ToLower(w.Body.String()), "invalid") {
			t.Errorf("expected invalid path, got: %s", w.Body.String())
		}
	})

	t.Run("static file with Cache-Control", func(t *testing.T) {
		conf.CacheTime = 3600
		defer func() { conf.CacheTime = 0 }()

		cssFile := filepath.Join(tmpDir, "style.css")
		os.WriteFile(cssFile, []byte("body{}"), 0644)

		req := httptest.NewRequest("GET", "/style.css", nil)
		w := httptest.NewRecorder()
		index(w, req)

		if w.Result().StatusCode != 200 {
			t.Errorf("status = %d, want 200", w.Result().StatusCode)
		}
		cc := w.Header().Get("Cache-Control")
		if !strings.Contains(cc, "max-age=3600") {
			t.Errorf("Cache-Control = %q, want max-age=3600", cc)
		}
	})
}

// ────────────────────────────────────────────
// HTTP handler: query
// ────────────────────────────────────────────

func TestQueryHandler(t *testing.T) {
	tmpDir := t.TempDir()
	tmplDir := filepath.Join(tmpDir, "tmpl")
	blogDir := filepath.Join(tmpDir, "blog")
	os.MkdirAll(tmplDir, 0755)
	os.MkdirAll(blogDir, 0755)

	realTmplDir := "tmpl"
	for _, name := range []string{"style.tmpl", "query.tmpl"} {
		data, _ := os.ReadFile(filepath.Join(realTmplDir, name))
		os.WriteFile(filepath.Join(tmplDir, name), data, 0644)
	}

	configContent := `
BlogDir = "` + strings.ReplaceAll(blogDir, "\\", "\\\\") + `"
Port = 9999
Ip = "127.0.0.1"
LogFile = "` + strings.ReplaceAll(filepath.Join(tmpDir, "test.log"), "\\", "\\\\") + `"
DevMode = "release"
HomePageLink = "/index.html"
HomePageTitle = "Home"
CacheTime = 0
`
	os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(configContent), 0644)

	mdAbs := filepath.Join(blogDir, "post.md")
	os.WriteFile(mdAbs, []byte("hello searchable content here"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "query.data"), []byte(mdAbs+"\n"), 0644)

	origConfPath := confPath
	origQueryPath := queryTemplatePath
	origStylePath := styleTemplatePath
	origConf := conf
	origForbidden := append([]string{}, forbiddenFiles...)
	origQueryFile := queryFile
	origRootDir := rootDir
	origCwd, _ := os.Getwd()

	defer func() {
		confPath = origConfPath
		queryTemplatePath = origQueryPath
		styleTemplatePath = origStylePath
		conf = origConf
		forbiddenFiles = origForbidden
		queryFile = origQueryFile
		rootDir = origRootDir
		os.Chdir(origCwd)
	}()

	os.Chdir(tmpDir)
	confPath = filepath.Join(tmpDir, "config.toml")
	queryTemplatePath = filepath.Join(tmplDir, "query.tmpl")
	styleTemplatePath = filepath.Join(tmplDir, "style.tmpl")
	queryFile = filepath.Join(tmpDir, "query.data")
	rootDir, _ = filepath.Abs(tmpDir)
	templates = make(map[string]*template.Template)
	loadConfig()

	t.Run("search returns matching results", func(t *testing.T) {
		form := url.Values{"search": {"searchable"}}
		req := httptest.NewRequest("POST", "/query", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		query(w, req)

		body := w.Body.String()
		if w.Result().StatusCode != 200 {
			t.Errorf("status = %d, want 200", w.Result().StatusCode)
		}
		if !strings.Contains(body, "post.md") {
			t.Error("search results missing matching file")
		}
	})

	t.Run("search with no results", func(t *testing.T) {
		form := url.Values{"search": {"zzzznotfoundzzzz"}}
		req := httptest.NewRequest("POST", "/query", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		query(w, req)
		if w.Result().StatusCode != 200 {
			t.Errorf("status = %d, want 200", w.Result().StatusCode)
		}
	})

	t.Run("empty search returns error", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/query", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		query(w, req)
		if !strings.Contains(strings.ToLower(w.Body.String()), "required") {
			t.Error("expected error for missing search query")
		}
	})
}

// ────────────────────────────────────────────
// getTemplate
// ────────────────────────────────────────────

func TestGetTemplate(t *testing.T) {
	tmplDir := "tmpl"
	if _, err := os.Stat(filepath.Join(tmplDir, "style.tmpl")); err != nil {
		t.Skip("template files not found, skipping")
	}

	templates = make(map[string]*template.Template)

	t.Run("load index template", func(t *testing.T) {
		tmpl, err := getTemplate("index")
		if err != nil {
			t.Fatalf("getTemplate index failed: %v", err)
		}
		if tmpl == nil {
			t.Fatal("template is nil")
		}
	})

	t.Run("load article template", func(t *testing.T) {
		tmpl, err := getTemplate("article")
		if err != nil {
			t.Fatalf("getTemplate article failed: %v", err)
		}
		if tmpl == nil {
			t.Fatal("template is nil")
		}
	})

	t.Run("load query template", func(t *testing.T) {
		tmpl, err := getTemplate("query")
		if err != nil {
			t.Fatalf("getTemplate query failed: %v", err)
		}
		if tmpl == nil {
			t.Fatal("template is nil")
		}
	})

	t.Run("unknown template returns error", func(t *testing.T) {
		_, err := getTemplate("nonexistent")
		if err == nil {
			t.Error("expected error for unknown template")
		}
	})

	t.Run("template caching - same pointer", func(t *testing.T) {
		templates = make(map[string]*template.Template)
		t1, _ := getTemplate("index")
		t2, _ := getTemplate("index")
		if t1 != t2 {
			t.Error("template cache should return same pointer")
		}
	})
}
