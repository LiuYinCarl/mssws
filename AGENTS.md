# AGENTS.md - MSSWS Development Guide

This document provides essential information for AI agents working with the MSSWS (Most Simple Static Web Server) codebase.

## Project Overview

MSSWS is a simple static web server written in Go that serves markdown files and PDFs with automatic rendering. It uses texme for markdown rendering, PDF.js for PDF preview, and provides full-text search functionality.

**Key Features:**
- Markdown file rendering with LaTeX support
- PDF file preview using embedded PDF.js
- Full-text search across markdown files
- RSS feed generation
- Directory monitoring for automatic index regeneration
- Simple configuration via TOML

## Essential Commands

### Build and Run

```bash
# Default: compile and run server
./run.sh

# Show help
./run.sh help

# Compile only
./run.sh compile

# Kill running server
./run.sh kill

# Restart server (kill and restart)
./run.sh restart

# Run tests
go test -v

# Build Docker image
docker build -t mssws .

# Run with Docker
docker run -d -p 8000:8000 -v $(pwd)/blog:/app/blog mssws
```

### Index Generation

```bash
# Generate index.html and query.data
./genindex.sh

# Note: run.sh automatically runs genindex.sh before starting
```

### Dependency Management

```bash
# Install Go dependencies (fixes missing imports)
go mod tidy
```

## Project Structure

```
mssws/
├── main.go                 # Main Go application
├── main_test.go           # Unit tests for main application
├── config.toml            # Configuration file
├── Dockerfile             # Docker configuration
├── run.sh                 # Build/run control script
├── genindex.sh           # Index generation script
├── genindex.py           # Python index/RSS generation
├── go.mod                # Go module dependencies
├── tmpl/                 # HTML templates
│   ├── index.tmpl
│   ├── article.tmpl
│   ├── query.tmpl
│   └── style.tmpl
├── lib/                  # Embedded libraries
│   ├── marked/          # Markdown parser
│   ├── texme/           # Markdown + LaTeX renderer
│   ├── mathjax/         # Math rendering
│   └── pdfjs/           # PDF viewer
└── image/               # Screenshots and images
```

## Configuration

The application is configured via `config.toml`:

```toml
# Server settings
Ip = "0.0.0.0"
Port = 8000
BlogDir = "./blog"          # Directory containing .md and .pdf files

# Site settings
SiteTitle = "www.man6.org"
HomePageLink = "/index.html"
FootPrint = ""              # Footer text (HTML allowed)

# RSS settings
RssFile = "rss.xml"
RssLink = "http://www.man6.org/rss.xml"
RssTitle = "RSS of man6.org"

# Development settings
DevMode = "release"         # "release" or "debug"
OpenDirMonitor = true       # Auto-regenerate index on file changes
CacheTime = 3600            # Cache timeout in seconds

# Site navigation links
[[SiteLinks]]
Title = "文档"
Url = "http://www.man6.org/doc.md"

[[SiteLinks]]
Title = "GitHub"
Url = "https://github.com/LiuYinCarl/mssws/"
```

## Development Workflow

### Setting Up Development Environment

1. **Prerequisites:**
   - Go 1.18+
   - Python 3
   - `tree` command (for genindex.sh)

2. **Initial Setup:**
   ```bash
   # Fix missing dependencies
   go mod tidy
   
   # Create blog directory (where content goes)
   mkdir -p blog
   
   # Add markdown/PDF files to blog directory
   cp myfile.md blog/
   ```

3. **Running in Debug Mode:**
   ```bash
   # Edit config.toml
   DevMode = "debug"
   
   # Run server
   ./run.sh
   ```

### File Requirements

- **Markdown files:** Must use `.md` extension (lowercase)
- **PDF files:** Must use `.pdf` extension (lowercase)
- **Filenames:** Should not contain spaces
- **Directory:** Content goes in `blog/` directory (configurable)

## Code Patterns and Conventions

### Go Code Structure

- **Main entry point:** `main.go`
- **Configuration:** Loaded via TOML using `github.com/pelletier/go-toml/v2`
- **File watching:** Uses `github.com/fsnotify/fsnotify` for directory monitoring
- **Templates:** Go's `text/template` package with .tmpl files
- **Logging:** Custom log functions with `info_log()`, `warn_log()`, `err_log()`

### Template System

- **Base template:** `style.tmpl` contains shared CSS and JavaScript
- **Page templates:** `index.tmpl`, `article.tmpl`, `query.tmpl` inherit from style.tmpl
- **Template variables:** Use Go template syntax `{{ .VariableName }}`

### File Processing Pipeline

1. **Index generation:** `genindex.sh` → `genindex.py` → creates `index.data`
2. **Content serving:** Go server reads markdown, wraps in texme template
3. **PDF serving:** Redirects to embedded PDF.js viewer
4. **Search:** Full-text search across markdown files using `query.data`

## Testing and Quality

### Current State
- **No unit tests** for main Go application
- **Embedded libraries** (marked, texme) have comprehensive test suites
- **CI/CD:** GitHub CodeQL analysis enabled for security scanning
- **Shell scripts:** Use shellcheck for linting (see README changelog)

### Recommended Testing Approach
```bash
# Manual testing workflow
1. Create test content in blog/ directory
2. Run ./run.sh
3. Access http://localhost:8000
4. Verify rendering, search, and PDF preview
```

## Common Issues and Solutions

### Missing Dependencies
```bash
# Symptoms: Import errors in main.go
# Fix:
go mod tidy
```

### File Not Found Errors
- Ensure `blog/` directory exists with proper permissions
- Check file extensions are `.md` or `.pdf` (lowercase)
- Verify filenames don't contain spaces

### Port Already in Use
```bash
# Kill existing instance
./run.sh kill
# Or manually find and kill process
pgrep mssws_prog
```

### Index Not Updating
- Check `OpenDirMonitor = true` in config.toml
- Run `./genindex.sh` manually to regenerate
- Verify `tree` command is installed

## Embedded Libraries

### texme (`lib/texme/`)
- Markdown + LaTeX rendering
- Version: See package.json in lib/texme/
- Configuration: Set via window.texme in style.tmpl

### marked (`lib/marked/`)
- Markdown parsing
- Version: See package.json in lib/marked/
- Used via CDN-style loading from `/lib/marked/marked.min.js`

### PDF.js (`lib/pdfjs/`)
- PDF rendering and preview
- Accessed via `/lib/pdfjs/web/viewer.html?file=path/to/file.pdf`

### MathJax (`lib/mathjax/`)
- LaTeX math rendering
- Loaded from `/lib/mathjax/es5/tex-mml-chtml.js`

## Deployment Notes

### Production Setup
1. Set `DevMode = "release"` in config.toml
2. Configure proper `SiteLink` for RSS generation
3. Set appropriate `CacheTime` for browser caching
4. Consider running behind reverse proxy (nginx, Apache)

### Content Management
- Add/remove files in `blog/` directory
- Run `./genindex.sh` after changes (or rely on directory monitor)
- RSS feed auto-generated with index update

### Security Considerations
- No admin interface (removed in 2023/09/06)
- `forbidden_files` list in code prevents access to certain files
- Static file serving only from configured directories

## Development History

Key changes from README:
- 2023/09/06: Removed admin page, added forbidden_files list
- 2023/12/10: Switched from config.json to config.toml
- 2024/06/02: Added log package for output management
- Various: Security fixes, library updates, feature additions

## Additional Resources

- **README.md**: Comprehensive documentation in English/Chinese
- **Example site**: http://www.man6.org/
- **GitHub repository**: https://github.com/LiuYinCarl/mssws/

---

*This document was auto-generated for AI agent assistance. Update as needed when project evolves.*