package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

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

	forbiddenFiles = make(map[string]bool)
	rootDir, _ = filepath.Abs("./")
)

func base_log(msg string) {
	if conf.DevMode == "debug" {
		fmt.Println(msg)
	}
	log.Println(msg)
}

func info_log(format string, v ... any) {
	msg := "[INFO] " + fmt.Sprintf(format, v...)
	base_log(msg)
}

func warn_log(format string, v ...any) {
	msg := "[WARN] " + fmt.Sprintf(format, v...)
	base_log(msg)
}

func err_log(format string, v ...any) {
	msg := "[ERROR] " + fmt.Sprintf(format, v...)
	base_log(msg)
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

	// Initialize forbidden files map
	forbiddenFiles["./directory_monitor.sh"] = true
	forbiddenFiles["./genindex.py"] = true
	forbiddenFiles["./main.go"] = true
	forbiddenFiles["./config.toml"] = true
	forbiddenFiles["./genindex.sh"] = true
	forbiddenFiles["./run.sh"] = true
	forbiddenFiles["./mssws_prog"] = true
	if conf.LogFile != "" {
		forbiddenFiles[conf.LogFile] = true
	}

	// Ensure BlogDir exists or can be created
	if _, err := os.Stat(conf.BlogDir); os.IsNotExist(err) {
		if err := os.MkdirAll(conf.BlogDir, 0755); err != nil {
			return fmt.Errorf("failed to create BlogDir %s: %w", conf.BlogDir, err)
		}
		info_log("created blog directory: %s", conf.BlogDir)
	}

	return nil
}

// 将目录及其子目录加入 fsnotify Watch
// fsnotify 默认不会监测子目录
func watchSubDir(watcher *fsnotify.Watcher, dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			path, err := filepath.Abs(path)
			if err != nil {
				err_log("get abs path failed, err:%v", err)
				return err
			}
			if err := watcher.Add(path); err != nil {
				err_log("watch path failed, path:%s, err:%v", path, err)
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
		err_log("NewWatcher failed, err:%v", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				info_log("run bash genindex.sh, event name:%s event op:%s", event.Name, event.Op)
				cmd := exec.Command("bash", "./genindex.sh")
				if err := cmd.Run(); err != nil {
					err_log("run genindex.sh failed, err:%v", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				err_log("watcher err:%v", err)
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

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func IsFile(path string) bool {
	return !IsDir(path)
}

// strings.Split 函数切割数组后可能会出现空字符串,违反直觉，这个函数用来去掉空字符串
func Split(s string, sep string) []string {
	tmp := strings.Split(s, sep)
	res := make([]string, 0)
	for _, k := range tmp {
		if k != "" {
			res = append(res, k)
		}
	}
	return res
}

func GetContentType(suffix string) string {
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

func query_single_file(filePath string, queryStr string) bool {
	if filePath == "" {
		err_log("query filepath is empty, filepath:%s, query_str:%s", filePath, queryStr)
		return false
	}

	// Validate the file path first
	if !validateQueryPath(filePath) {
		err_log("query path validation failed: %s", filePath)
		return false
	}

	if ok := IsFile(filePath); !ok {
		err_log("query path is not a file, filepath:%s, query_str:%s", filePath, queryStr)
		return false
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		err_log("failed to read file for query: %s, error: %v", filePath, err)
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
		err_log("failed to parse form: %v", err)
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

	info_log("search query: %s", query_str)

	f_query, err := os.Open(queryFile)
	if err != nil {
		err_log("query open file failed, query_str:%s, queryFile:%s", query_str, queryFile)
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

		if ok := query_single_file(filepath, query_str); ok {
			str := fmt.Sprintf("<a href=\"%s\">%s</a></br>", line, line)
			return_lines = append(return_lines, str)
		}
	}

	var buffer bytes.Buffer
	for _, s := range return_lines {
		buffer.WriteString(s)
	}

	temp, err := template.ParseFiles(queryTemplatePath, styleTemplatePath)
	if err != nil {
		err_log("load query template failed, queryTemplatePath:%s, styleTemplatePath:%s",
			queryTemplatePath, styleTemplatePath)
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
		err_log("open index.data failed")
		w.Write([]byte("Sorry, Index Page Not Exist."))
		return
	}

	content_type := GetContentType("html")
	w.Header().Set("Content-Type", content_type)

	temp, err := template.ParseFiles(indexTemplatePath, styleTemplatePath)
	if err != nil {
		err_log("load index template failed, queryTemplatePath:%s, styleTemplatePath:%s",
			queryTemplatePath, styleTemplatePath)
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
		err_log("url decode failed, path:%s, err:%v", rawPath, err)
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
		err_log("path validation failed: %v", err)
		w.Write([]byte("Invalid path requested."))
		return
	}

	info_log("visit file:%s", filePath)

	// Check forbidden files
	if _, ok := forbiddenFiles[filePath]; ok {
		w.Write([]byte("Access to this file is forbidden."))
		return
	}

	// Verify the file exists and is a regular file
	if ok := IsFile(filePath); !ok {
		w.Write([]byte("[404] File not found."))
		return
	}

	suffix := ""
	if split_list := Split(filePath, "."); len(split_list) > 1 {
		suffix = split_list[len(split_list)-1]
	}
	content_type := GetContentType(suffix)
	w.Header().Set("Content-Type", content_type)

	if conf.CacheTime > 0 {
		if suffix == "js" || suffix == "css" || suffix == "ico" {
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, public", conf.CacheTime))
		}
	}

	// markdown file
	if suffix == "md" {
		temp, err := template.ParseFiles(articleTemplatePath, styleTemplatePath)
		if err != nil {
			err_log("load article template failed, queryTemplatePath:%s, styleTemplatePath:%s",
				queryTemplatePath, styleTemplatePath)
			w.Write([]byte("load article template file failed."))
			return
		}

		content, err := os.ReadFile(filePath)
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
			warn_log("trying to visit non-existent file:%s", filePath)
			w.Write([]byte("404 file not exist."))
			return
		}
		w.Write(content)
	}
}

func main() {
	if err := loadConfig(); err != nil {
		err_log("Failed to load configuration: %v", err)
		os.Exit(1)
	}
	info_log("mssws starting...")

	// init log
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log_file, err := os.OpenFile(conf.LogFile, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0644)
	if err != nil {
		err_log("open log file failed, file=%s, err=%v", conf.LogFile, err)
		os.Exit(1)
	}
	log.SetOutput(log_file)
	info_log("create logger success")

	if conf.OpenDirMonitor == true {
		go dirMonitor()
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/query", query)

	ip_port := conf.Ip + ":" + strconv.Itoa(conf.Port)
	info_log(fmt.Sprintf("mssws start listen and serve in %s...", ip_port))

	err = http.ListenAndServe(ip_port, nil)
	if err != nil {
		err_log("listen and serve failed, err=%v", err)
		os.Exit(1)
	}
}
