package main

import (
	"bufio"
	"bytes"
	"fmt"
	"text/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"strconv"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/fsnotify/fsnotify"
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

// 加载 toml 配置
func loadConfig() bool {
	// forbidden visit files
	forbidden_files["./directory_monitor.sh"] = true
	forbidden_files["./genindex.py"]          = true
	forbidden_files["./main.go"]              = true
	forbidden_files["./server.log"]           = true
	forbidden_files["./config.toml"]          = true
	forbidden_files["./genindex.sh"]          = true
	forbidden_files["./run.sh"]               = true
	forbidden_files["./mssws_prog"]           = true

	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		fmt.Printf("read config file failed. file path is %s", confPath)
		return false
	}

	err = toml.Unmarshal(data, &conf)
	if err != nil {
		fmt.Printf("config.toml's content is error, %s", err)
		return false
	}

	return true
}

// 将目录及其子目录加入 fsnotify Watch
// fsnotify 默认不会监测子目录
func watchSubDir(watcher *fsnotify.Watcher, dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			path, err := filepath.Abs(path)
			if err != nil {
				log.Println(err)
				return err
			}
			if err := watcher.Add(path); err != nil {
				log.Printf("watch %s failed, err: ", path, err)
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
		log.Fatal("NewWatcher failed: ", err)
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
				log.Printf("%s %s\n", event.Name, event.Op)
				log.Println("run bash genindex.sh")
				cmd := exec.Command("bash", "./genindex.sh")
				if err := cmd.Run(); err != nil {
					log.Printf("run genindex.sh failed, err: ", err)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error: ", err)
			}
		}
	}()

	watchSubDir(watcher, conf.BlogDir)
	<-done
}


func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
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
	// return "text/html;charset=utf-8"
}

func query_single_file(filepath string, query_str string) bool {
	if filepath == "" {
		return false
	}

	if ok := IsFile(filepath); !ok {
		return false
	}

	os_cmd := exec.Command("grep", query_str, filepath)
	// create command stdout pipe
	stdout, err := os_cmd.StdoutPipe()
	if err != nil {
		return false
	}

	// run command
	if err := os_cmd.Start(); err != nil {
		return false
	}

	// read command output
	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return false
	}

	out_str := strings.TrimSpace(string(bytes))
	if out_str == "" {
		return false
	} else {
		return true
	}
}

func query(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["search"]) == 0 {
		w.Write([]byte("query string is empty."))
		return
	}

	query_str := r.Form["search"][0]
	if query_str == "" {
		w.Write([]byte("query string is empty."))
		return
	}

	f_query, err := os.Open(query_file)
	if err != nil {
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

		if ok := query_single_file(string(line), query_str); ok {
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
		w.Write([]byte("load query template file failed."))
		return
	}

	article := Article{
		SiteTitle:             "",
		HomePageLink:          conf.HomePageLink,
		HomePageTitle:         conf.HomePageTitle,
		FootPrint:             "",
		Content:               buffer.String(),
	}
	temp.Execute(w, article)
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadFile("index.data")
	if err != nil {
		w.Write([]byte("Sorry, Index Page Not Exist."))
		return
	}

	content_type := GetContentType("html")
	w.Header().Set("Content-Type", content_type)

	temp, err := template.ParseFiles(indexTemplatePath, styleTemplatePath)
	if err != nil {
		w.Write([]byte("load index template file failed."))
		return
	}

	index := Index{
		SiteTitle: conf.SiteTitle,
		FootPrint: conf.FootPrint,
		Content: string(content),
		SiteLinks: conf.SiteLinks,
	}

	temp.Execute(w, index)
}

func index(w http.ResponseWriter, r *http.Request) {
	url, err := url.PathUnescape(r.URL.Path)
	if err != nil {
		w.Write([]byte("url decode error."))
		return
	}

	if url == "/" || url == "" || strings.ToLower(url) == "/index.html" {
		indexPage(w, r)
		return
	}

	url = strings.TrimSpace(url)
	filePath := fmt.Sprintf(".%s", url)

	fmt.Println("filePath: ", filePath)

	if _, ok := forbidden_files[filePath]; ok {
		w.Write([]byte("try to visit forbieedn file."))
		return
	}
	real_path, err := filepath.Abs(filePath)
	if err != nil {
		w.Write([]byte("invalid link."))
		return
	}
	if !filepath.HasPrefix(real_path, root_dir) {
		w.Write([]byte("try visit invalid directory."))
		return
	}
	if filepath.HasPrefix(real_path, root_dir+"/.git") {
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

	// markdown file
	if filePath[len(filePath)-3:] == ".md" {
		temp, err := template.ParseFiles(articleTemplatePath, styleTemplatePath)
		if err != nil {
			w.Write([]byte("load article template file failed."))
			return
		}

		// content := Md2html(filePath)
		content, err := ioutil.ReadFile(filePath)
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
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Println("unexists = ", filePath)
			w.Write([]byte("404 file not exist."))
			return
		}
		w.Write(content)
	}
}


func main() {
	loadConfig()

	if conf.OpenDirMonitor == true {
		go dirMonitor()
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/query", query)

	ip_port := conf.Ip + ":" + strconv.Itoa(conf.Port)
	log.Fatal(http.ListenAndServe(ip_port, nil))
}
