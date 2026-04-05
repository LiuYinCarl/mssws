package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/fsnotify/fsnotify"
	toml "github.com/pelletier/go-toml/v2"
)

type SiteLink struct {
	Title string
	Url   string
}

type config struct {
	SiteTitle             string
	HomePageLink          string
	HomePageTitle         string
	FootPrint             string
	BlogDir               string
	Ip                    string
	Port                  int
	OpenDirMonitor        bool
	SiteLinks             []SiteLink
	CacheTime             int
	LogFile               string
	DevMode               string
}

type Article struct {
	SiteTitle             string
	HomePageLink          string
	HomePageTitle         string
	FootPrint             string
	Content               string
}

type Index struct {
	SiteTitle string
	FootPrint string
	Content string
	SiteLinks []SiteLink
}

var (
	conf config
	confPath            = "./config.toml"
	indexTemplatePath   = "./tmpl/index.tmpl"
	articleTemplatePath = "./tmpl/article.tmpl"
	queryTemplatePath   = "./tmpl/query.tmpl"
	styleTemplatePath   = "./tmpl/style.tmpl"
	queryFile           = "query.data"

	// Forbidden files that should not be served
	forbiddenFiles = []string{
		"./directory_monitor.sh",
		"./genindex.py",
		"./main.go",
		"./config.toml",
		"./genindex.sh",
		"./run.sh",
		"./mssws_prog",
	}

	rootDir, _ = filepath.Abs("./")

	// Template cache
	templates = make(map[string]*template.Template)
)

func baseLog(msg string) {
	if conf.DevMode == "debug" {
		fmt.Println(msg)
	}
	log.Println(msg)
}

func infoLog(format string, v ... any) {
	msg := "[INFO] " + fmt.Sprintf(format, v...)
	baseLog(msg)
}

func warnLog(format string, v ...any) {
	msg := "[WARN] " + fmt.Sprintf(format, v...)
	baseLog(msg)
}

func errLog(format string, v ...any) {
	msg := "[ERROR] " + fmt.Sprintf(format, v...)
	baseLog(msg)
}

// loadConfig loads and validates the TOML configuration
func loadConfig() error {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("read config file failed: %w", err)
	}

	if err := toml.Unmarshal(data, &conf); err != nil {
		return fmt.Errorf("parse config failed: %w", err)
	}

	// Validate required configuration fields
	if conf.BlogDir == "" {
		return errors.New("BlogDir is required in config")
	}
	if conf.LogFile == "" {
		return errors.New("LogFile is required in config")
	}
	if conf.Port <= 0 || conf.Port > 65535 {
		return fmt.Errorf("invalid Port: %d (must be 1-65535)", conf.Port)
	}

	// Add log file to forbidden files if specified
	if conf.LogFile != "" {
		forbiddenFiles = append(forbiddenFiles, conf.LogFile)
	}

	// Ensure BlogDir exists or can be created
	if _, err := os.Stat(conf.BlogDir); os.IsNotExist(err) {
		if err := os.MkdirAll(conf.BlogDir, 0755); err != nil {
			return fmt.Errorf("failed to create BlogDir %s: %w", conf.BlogDir, err)
		}
		infoLog("created blog directory: %s", conf.BlogDir)
	}

	return nil
}

// 将目录及其子目录加入 fsnotify Watch
// fsnotify 默认不会监测子目录
func watchSubDir(watcher *fsnotify.Watcher, dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errLog("walk error: %v", err)
			return err
		}
		if info.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				errLog("get abs path failed, err:%v", err)
				return err
			}
			if err := watcher.Add(absPath); err != nil {
				errLog("watch path failed, path:%s, err:%v", absPath, err)
				return err
			}
		}
		return nil
	})
}

// 文件监控
func dirMonitor() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errLog("NewWatcher failed, err:%v", err)
		return
	}
	defer watcher.Close()

	// 防抖定时器
	var debounceTimer *time.Timer
	var debounceMutex sync.Mutex
	debounceDuration := 500 * time.Millisecond

	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				infoLog("file event: %s %s", event.Name, event.Op)

				// 防抖处理
				debounceMutex.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					infoLog("run bash genindex.sh after debounce")
					cmd := exec.Command("bash", "./genindex.sh")
					if err := cmd.Run(); err != nil {
						errLog("run genindex.sh failed, err:%v", err)
					}
				})
				debounceMutex.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				errLog("watcher err:%v", err)
			}
		}
	}()

	watchSubDir(watcher, conf.BlogDir)
	<-done
}

// validatePath validates and sanitizes a requested file path
func validatePath(requestPath string) (string, error) {
	if requestPath == "" || requestPath == "/" {
		return ".", nil
	}

	// Remove any leading slash and clean the path
	cleanPath := strings.TrimPrefix(requestPath, "/")
	if cleanPath == "" {
		return ".", nil
	}

	// Use filepath.Clean to remove any ".." or "." components
	cleanPath = filepath.Clean(cleanPath)

	// Ensure the path is within the root directory
	absPath, err := filepath.Abs(filepath.Join(".", cleanPath))
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	root, _ := filepath.Abs(".")
	if !strings.HasPrefix(absPath, root) {
		return "", errors.New("path traversal attempt detected")
	}

	// Check for forbidden directories (e.g., .git)
	if strings.Contains(absPath, "/.git/") || strings.HasSuffix(absPath, "/.git") {
		return "", errors.New("access to git directory forbidden")
	}

	return filepath.Join(".", cleanPath), nil
}

func isDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func isFile(path string) bool {
	return !isDir(path)
}

// strings.Split 函数切割数组后可能会出现空字符串,违反直觉，这个函数用来去掉空字符串
func split(s string, sep string) []string {
	tmp := strings.Split(s, sep)
	res := make([]string, 0)
	for _, k := range tmp {
		if k != "" {
			res = append(res, k)
		}
	}
	return res
}

func getContentType(suffix string) string {
	switch suffix {
	case "html":
		return "text/html;charset=utf-8"
	case "xml":
		return "application/rss+xml;charset=utf-8"
	case "ico":
		return "image/x-icon"
	case "js":
		return "application/x-javascript"
	case "css":
		return "text/css"
	case "pdf":
		return "application/pdf"
	case "png":
		return "application/x-png"
	case "svg":
		return "image/svg+xml"
	case "ttf":
		return "application/x-font-truetype"
	case "woff", "woff2":
		return "application/x-font-woff"
	default:
		return "text/html;charset=utf-8"
	}
}

// validateQueryPath validates that a file path is safe for query operations
func validateQueryPath(filePath string) bool {
	if filePath == "" {
		return false
	}

	// Ensure the path is within the blog directory
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	root, _ := filepath.Abs(".")
	if !strings.HasPrefix(absPath, root) {
		return false
	}

	// Check for path traversal attempts
	cleanPath := filepath.Clean(filePath)
	if cleanPath != filePath && !strings.HasPrefix(cleanPath, "./") {
		return false
	}

	// Ensure it's a markdown file
	if !strings.HasSuffix(strings.ToLower(filePath), ".md") {
		return false
	}

	return true
}

func querySingleFile(filePath string, queryStr string) bool {
	if filePath == "" {
		errLog("query filepath is empty, filepath:%s, query_str:%s", filePath, queryStr)
		return false
	}

	// Validate the file path first
	if !validateQueryPath(filePath) {
		errLog("query path validation failed: %s", filePath)
		return false
	}

	if ok := isFile(filePath); !ok {
		errLog("query path is not a file, filepath:%s, query_str:%s", filePath, queryStr)
		return false
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		errLog("failed to read file for query: %s, error: %v", filePath, err)
		return false
	}

	// Simple validation: query string should not be empty or too long
	if queryStr == "" || len(queryStr) > 1000 {
		return false
	}

	return strings.Contains(string(content), queryStr)
}

// sanitizeSearchString removes potentially dangerous characters from search query
func sanitizeSearchString(input string) string {
	// Remove null bytes, control characters, and excessive whitespace
	input = strings.TrimSpace(input)

	// Limit length to prevent abuse
	if len(input) > 1000 {
		input = input[:1000]
	}

	// Remove potentially dangerous patterns (basic protection)
	dangerousPatterns := []string{
		"../", "..\\", "/etc/", "/proc/", "/dev/",
		";", "|", "&", "$", "`", "\"", "'",
		"<script>", "</script>", "javascript:", "onload=",
	}

	for _, pattern := range dangerousPatterns {
		input = strings.ReplaceAll(input, pattern, "")
	}

	return input
}

func query(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		errLog("failed to parse form: %v", err)
		w.Write([]byte("Invalid form data."))
		return
	}

	searchValues := r.Form["search"]
	if len(searchValues) == 0 {
		w.Write([]byte("Search query is required."))
		return
	}

	query_str := searchValues[0]
	if query_str == "" {
		w.Write([]byte("Search query cannot be empty."))
		return
	}

	// Sanitize the search query
	query_str = sanitizeSearchString(query_str)
	if query_str == "" {
		w.Write([]byte("Invalid search query."))
		return
	}

	infoLog("search query: %s", query_str)

	f_query, err := os.Open(queryFile)
	if err != nil {
		errLog("query open file failed, query_str:%s, queryFile:%s", query_str, queryFile)
		w.Write([]byte("open query file error."))
		return
	}
	defer f_query.Close()

	var return_lines []string

	br := bufio.NewReader(f_query)
	for {
		line, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}

		filepath := string(line)

		if ok := querySingleFile(filepath, query_str); ok {
			str := fmt.Sprintf("<a href=\"%s\">%s</a></br>", line, line)
			return_lines = append(return_lines, str)
		}
	}

	var buffer bytes.Buffer
	for _, s := range return_lines {
		buffer.WriteString(s)
	}

	temp, err := getTemplate("query")
	if err != nil {
		errLog("load query template failed: %v", err)
		w.Write([]byte("load query template file failed."))
		return
	}

	article := Article{
		SiteTitle:     "",
		HomePageLink:  conf.HomePageLink,
		HomePageTitle: conf.HomePageTitle,
		FootPrint:     "",
		Content:       buffer.String(),
	}
	temp.Execute(w, article)
}

func indexPage(w http.ResponseWriter, _ *http.Request) {
	content, err := os.ReadFile("index.data")
	if err != nil {
		errLog("open index.data failed")
		w.Write([]byte("Sorry, Index Page Not Exist."))
		return
	}

	content_type := getContentType("html")
	w.Header().Set("Content-Type", content_type)

	temp, err := getTemplate("index")
	if err != nil {
		errLog("load index template failed: %v", err)
		w.Write([]byte("load index template file failed."))
		return
	}

	index := Index{
		SiteTitle: conf.SiteTitle,
		FootPrint: conf.FootPrint,
		Content:   string(content),
		SiteLinks: conf.SiteLinks,
	}

	temp.Execute(w, index)
}

func index(w http.ResponseWriter, r *http.Request) {
	rawPath := r.URL.Path
	unescapedPath, err := url.PathUnescape(rawPath)
	if err != nil {
		errLog("url decode failed, path:%s, err:%v", rawPath, err)
		w.Write([]byte("url decode error."))
		return
	}

	// Handle index page
	if unescapedPath == "/" || unescapedPath == "" || strings.ToLower(unescapedPath) == "/index.html" {
		indexPage(w, r)
		return
	}

	// Validate and sanitize the file path
	filePath, err := validatePath(unescapedPath)
	if err != nil {
		errLog("path validation failed: %v", err)
		w.Write([]byte("Invalid path requested."))
		return
	}

	infoLog("visit file:%s", filePath)

	// Check forbidden files
	isForbidden := false
	for _, forbidden := range forbiddenFiles {
		if filePath == forbidden {
			isForbidden = true
			break
		}
	}
	if isForbidden {
		w.Write([]byte("Access to this file is forbidden."))
		return
	}

	// Verify the file exists and is a regular file
	if ok := isFile(filePath); !ok {
		w.Write([]byte("[404] File not found."))
		return
	}

	suffix := ""
	if split_list := split(filePath, "."); len(split_list) > 1 {
		suffix = split_list[len(split_list)-1]
	}
	content_type := getContentType(suffix)
	w.Header().Set("Content-Type", content_type)

	if conf.CacheTime > 0 {
		if suffix == "js" || suffix == "css" || suffix == "ico" {
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", conf.CacheTime))
		}
	}

	// markdown file
	if suffix == "md" {
		temp, err := getTemplate("article")
		if err != nil {
			errLog("load article template failed: %v", err)
			w.Write([]byte("load article template file failed."))
			return
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			errLog("read markdown file failed, path:%s, err:%v", filePath, err)
			w.Write([]byte("read file error."))
			return
		}
		articleName := path.Base(filePath)

		article := Article{
			SiteTitle:             articleName,
			HomePageLink:          conf.HomePageLink,
			HomePageTitle:         conf.HomePageTitle,
			FootPrint:             conf.FootPrint,
			Content:               string(content),
		}

		temp.Execute(w, article)
	} else {
		content, err := os.ReadFile(filePath)
		if err != nil {
			warnLog("trying to visit non-existent file:%s", filePath)
			w.Write([]byte("404 file not exist."))
			return
		}
		w.Write(content)
	}
}

// getTemplate loads and caches templates
func getTemplate(name string) (*template.Template, error) {
	if t, ok := templates[name]; ok {
		return t, nil
	}

	var t *template.Template
	var err error

	switch name {
	case "index":
		t, err = template.ParseFiles(indexTemplatePath, styleTemplatePath)
	case "article":
		t, err = template.ParseFiles(articleTemplatePath, styleTemplatePath)
	case "query":
		t, err = template.ParseFiles(queryTemplatePath, styleTemplatePath)
	default:
		return nil, fmt.Errorf("unknown template: %s", name)
	}

	if err != nil {
		return nil, err
	}

	templates[name] = t
	return t, nil
}

func main() {
	if err := loadConfig(); err != nil {
		errLog("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	infoLog("mssws starting...")

	// init log
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log_file, err := os.OpenFile(conf.LogFile, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0644)
	if err != nil {
		errLog("open log file failed, file=%s, err=%v", conf.LogFile, err)
		os.Exit(1)
	}
	log.SetOutput(log_file)
	infoLog("create logger success")

	if conf.OpenDirMonitor == true {
		go dirMonitor()
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/query", query)

	ip_port := conf.Ip + ":" + strconv.Itoa(conf.Port)
	infoLog(fmt.Sprintf("mssws start listen and serve in %s...", ip_port))

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         ip_port,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		infoLog("shutting down server...")

		// 创建5秒超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 关闭HTTP服务器
		if err := server.Shutdown(ctx); err != nil {
			errLog("server shutdown failed: %v", err)
		}

		infoLog("server stopped")
	}()

	// 启动服务器
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		errLog("listen and serve failed, err=%v", err)
		os.Exit(1)
	}
}
