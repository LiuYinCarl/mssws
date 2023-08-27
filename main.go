package main

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	"strings"
	"time"
)

type SiteLink struct {
	Title string `json:"Title"`
	Url   string `json: "Url"`
}

type config struct {
	SiteTitle             string `json:"SiteTitle"`
	SiteLink			  string `json:"SiteLink"`
	HomePageLink          string `json:"HomePageLink"`
	HomePageTitle         string `json:"HomePageTitle"`
	FootPrint             string `json:"FootPrint"`
	TexmeCDNLink          string `json:"TexmeCDNLink"`
	BlogDir               string `json:"BlogDir"`
	IP                    string `json:"IP"`
	Port                  string `json:"Port"`
	OpenDirMonitor        string `json:"OpenDirMonitor"`
	MonitorScript         string `json:"MonitorScript"`
	MonitorRefreshTick    time.Duration    `json:"MonitorRefreshTick"`
	SiteLinks             []SiteLink `json:"SiteLinks"`
}

type Article struct {
	SiteTitle             string
	HomePageLink          string
	HomePageTitle         string
	FootPrint             string
	TexmeCDNLink          string
	Content               string
}

type Index struct {
	SiteTitle string
	FootPrint string
	TexmeCDNLink string
	Content string
	SiteLinks []SiteLink
}


var (
	conf config
	is_head				= true
	confPath			= "./config.json"
	indexTemplatePath	= "./index_template.html"
	articleTemplatePath = "./article_template.html"
	queryTemplatePath	= "./query_template.html"
	styleTemplatePath   = "./style.tmpl"
	query_file			= "query.data"
	admin_script		= "./admin.sh"
)

// 加载 json 配置
func loadConfig() bool {
	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		fmt.Printf("read config file failed. file path is %s", confPath)
		return false
	}

	err = json.Unmarshal(data, &conf)
	if err != nil {
		fmt.Printf("config.json's content is error")
		return false
	}

	conf.TexmeCDNLink			= conf.SiteLink + conf.TexmeCDNLink
	return true
}

// 文件监控
func dirMonitor() {
	c := time.Tick(conf.MonitorRefreshTick * time.Second)
	for _ = range c {
		cmd := exec.Command("bash", conf.MonitorScript)
		if err := cmd.Run(); err != nil {
			fmt.Printf("run dir monitorScript:%s failed\n", conf.MonitorScript)
		}
	}
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

func admin(w http.ResponseWriter, r *http.Request) {
	url, err := url.PathUnescape(r.URL.Path)
	if err != nil {
		w.Write([]byte("url decode error."))
		return
	}

	params := strings.Split(url, "/")
	// remove first empty string
	if params[0] == "" {
		params = params[1:]
	}
	// remove "admin" field
	// ["admin", "password", "update"] => ["password", "update"]
	params = params[1:]
	fmt.Println(params)

	admin_cmd := admin_script + " " + fmt.Sprintf(strings.Join(params, " "))
	os_cmd := exec.Command("/bin/bash", "-c", admin_cmd)

	stdout, err := os_cmd.StdoutPipe()
	if err != nil {
		w.Write([]byte("[step 1] run admin command failed."))
		return
	}

	// run command
	if err := os_cmd.Start(); err != nil {
		w.Write([]byte("[step 2]run admin command failed."))
		fmt.Println(err)
		return
	}

	// read admin script output and return to user
	bytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		w.Write([]byte("[step 3] run admin command failed."))
		return
	}

	w.Write(bytes)
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
		TexmeCDNLink:          conf.TexmeCDNLink,
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
		TexmeCDNLink: conf.TexmeCDNLink,
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

	fmt.Printf("filePath: %s\n", filePath)

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
			TexmeCDNLink:          conf.TexmeCDNLink,
			Content:               string(content),
		}

		temp.Execute(w, article)
	} else {
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			w.Write([]byte("404 file not exist.."))
			return
		}
		w.Write(content)
	}
}


func main() {
	loadConfig()

	if conf.OpenDirMonitor == "true" {
		go dirMonitor()
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/query", query)
	http.HandleFunc("/admin/", admin)

	ip_port := conf.IP + ":" + conf.Port
	log.Fatal(http.ListenAndServe(ip_port, nil))
}
