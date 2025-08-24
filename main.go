package main

import (
	"bufio"
	"bytes"
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
	is_head				= true
	confPath			= "./config.toml"
	indexTemplatePath	= "./tmpl/index.tmpl"
	articleTemplatePath = "./tmpl/article.tmpl"
	queryTemplatePath	= "./tmpl/query.tmpl"
	styleTemplatePath   = "./tmpl/style.tmpl"
	query_file			= "query.data"

	forbidden_files = make(map[string]bool)
	root_dir, _ = filepath.Abs("./")
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

// 加载 toml 配置
func loadConfig() bool {
	data, err := os.ReadFile(confPath)
	if err != nil {
		err_log("read config file failed. file path is %s", confPath)
		return false
	}

	err = toml.Unmarshal(data, &conf)
	if err != nil {
		err_log("config.toml's content is error, %s", err)
		return false
	}

	// forbidden visit files
	forbidden_files["./directory_monitor.sh"] = true
	forbidden_files["./genindex.py"]          = true
	forbidden_files["./main.go"]              = true
	forbidden_files["./config.toml"]          = true
	forbidden_files["./genindex.sh"]          = true
	forbidden_files["./run.sh"]               = true
	forbidden_files["./mssws_prog"]           = true
	forbidden_files[conf.LogFile]             = true

	return true
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

func query_single_file(filepath string, query_str string) bool {
	if filepath == "" {
		err_log("query filepath is empty, filepath:%s, quert_str:%s", filepath, query_str)
		return false
	}

	if ok := IsFile(filepath); !ok {
		err_log("query path is not file, filepath:%s, query_str:%s", filepath, query_str)
		return false
	}

	content, err := os.ReadFile(filepath)
	if err != nil {
		return false
	}

	return strings.Contains(string(content), query_str)
}

func query(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	info_log("query string is:%s", r.Form["search"])
	if len(r.Form["search"]) == 0 {
		w.Write([]byte("query string is empty."))
		return
	}

	query_str := r.Form["search"][0]
	if query_str == "" {
		err_log("query string is empty")
		w.Write([]byte("query string is empty."))
		return
	}

	f_query, err := os.Open(query_file)
	if err != nil {
		err_log("query open file failed, query_str:%s, query_file:%s", query_str, query_file)
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
		err_log("load query template failed, quertTemplatePath:%s, stypeTemplatePath:%s",
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
		err_log("load index template failed, quertTemplatePath:%s, stypeTemplatePath:%s",
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
	url, err := url.PathUnescape(r.URL.Path)
	if err != nil {
		err_log("url decode failed, path:%s, err:%v", r.URL.Path, err)
		w.Write([]byte("url decode error."))
		return
	}

	if url == "/" || url == "" || strings.ToLower(url) == "/index.html" {
		indexPage(w, r)
		return
	}

	url = strings.TrimSpace(url)
	filePath := fmt.Sprintf(".%s", url)

	info_log("visit file:%s", filePath)

	if _, ok := forbidden_files[filePath]; ok {
		w.Write([]byte("try to visit forbieedn file."))
		return
	}
	real_path, err := filepath.Abs(filePath)
	if err != nil {
		w.Write([]byte("invalid link."))
		return
	}
	if !strings.HasPrefix(real_path, root_dir) {
		w.Write([]byte("try visit invalid directory."))
		return
	}
	if strings.HasPrefix(real_path, root_dir+"/.git") {
		w.Write([]byte("try to visit forbidden directories."))
		return
	}

	if ok := IsFile(filePath); !ok {
		w.Write([]byte("[404] file not exist."))
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
			err_log("load article template failed, quertTemplatePath:%s, stypeTemplatePath:%s",
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
			warn_log("try to visit unexist file:%s", filePath)
			w.Write([]byte("404 file not exist."))
			return
		}
		w.Write(content)
	}
}

func main() {
	loadConfig()
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
