package main

import (
    "log"
    "net/http"
    "os"
    "fmt"
    "strings"
    "bufio"
    "bytes"
    "io"
    "io/ioutil"
    "os/exec"
    "net/url"
)

var is_head = true


func Exists(path string) bool {
	_, err := os.Stat(path)    //os.Stat获取文件信息
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
        return "text/xml"
    case "ttf":
        return "application/x-font-truetype"
    case "woff":
    case "woff2":
        return "application/x-font-woff"
    default:
        return "text/html;charset=utf-8"
    }
    // not used, but Go want me add it.
    return "text/html;charset=utf-8"
}

func TransLine(line string) string {
    if ok := strings.Contains(line, "```"); ok {
        if strings.Index(line, "```") != 0{
            return line
        }

        line := strings.TrimSpace(line)
        if len(line) == 3 {
            if is_head {
                ret := "<pre><code class=\"\">"
                is_head = false
                return ret
            } else {
                ret := "</code></pre>"
                is_head = true
                return ret
            }
        } else {
            is_head = false
            return fmt.Sprintf("<pre><code class=\"%s\">", line[3:])               
        }
    }
    return line
}

func GetHead(title string) string {
    html := `<!DOCTYPE html><script src="https://cdn.jsdelivr.net/npm/texme@0.9.0"></script>
            <head>
            <meta charset="UTF-8">
            <title>` + title +  `</title>
            <link href="https://cdn.bootcss.com/highlight.js/9.12.0/styles/atom-one-dark.min.css" rel="stylesheet">
            <script src="https://cdn.bootcss.com/highlight.js/9.12.0/highlight.min.js"></script>
          <script >hljs.initHighlightingOnLoad();</script> 
          </head>`

          return html + "<a href=\"http://www.bearcarl.top/index.html\">Home Page</a>\n\n"
}

func Md2html(file_path string) []byte {
    var new_lines []string

    title := file_path[:len(file_path)-3]
    html := GetHead(title)
    new_lines = append(new_lines, html)

    md_file, err := os.Open(file_path)
    if err != nil {
        return []byte("internal error")
    }
    defer md_file.Close()

    br := bufio.NewReader(md_file)
    for {
        line, _, c := br.ReadLine()
        if c == io.EOF {
            break
        }
        
        l := TransLine(string(line)) + "\n"
        new_lines = append(new_lines, l)
    }


    var buffer bytes.Buffer
    for _, s := range new_lines {
        buffer.WriteString(s)
    }

    return buffer.Bytes()
}

var query_result_head = `
    <html>
    <!DOCTYPE html><script src="https://cdn.jsdelivr.net/npm/texme@0.9.0"></script>
    <head>
    <title>Query Result</title>
    </head>
    <body>
    <div>
    the keyword was found in the following files:
    </div>
    <hr>
    `
var query_result_tail = `
    </body>
    </html>
    `
var query_file = "query.data"

func query_single_file(filepath string, query_str string) bool {
    if filepath == "" {
        return false
    }

    if ok := IsFile(filepath); !ok {
        // todo: print log
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
    return_lines = append(return_lines, query_result_head)

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

    return_lines = append(return_lines, query_result_tail)

    var buffer bytes.Buffer
    for _, s := range return_lines {
        buffer.WriteString(s)
    }

    w.Write(buffer.Bytes())
}

func index(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/" || r.URL.Path == "" {
        content, err := ioutil.ReadFile("index.html")
        if err != nil {
            w.Write([]byte("index.html no exist."))
            return
        }
        w.Write(content)
        return
    }

    url, err := url.PathUnescape(r.URL.Path)
    if err != nil {
        w.Write([]byte("url decode error."))
        return
    }
    url = strings.TrimSpace(url)
    path := fmt.Sprintf(".%s", url)
    
    if ok := IsFile(path); !ok {
        w.Write([]byte("404 file not exist."))
        return
    }
    

    suffix := ""
    if split_list := Split(path, "."); len(split_list) > 1 {
        suffix = split_list[len(split_list)-1]
    }
    content_type := GetContentType(suffix)
    w.Header().Set("Content-Type", content_type)

    // markdown file
    if path[len(path)-3:] == ".md" {
        content := Md2html(path)
        w.Write(content)
    } else {
        content, err := ioutil.ReadFile(path)
        if err != nil {
            w.Write([]byte("404 file not exists."))
            return
        }
        w.Write(content)
    }
}


func main() {
    ip := "0.0.0.0"
    port := "80"

    http.HandleFunc("/", index)
    http.HandleFunc("/query", query)

    ip_port := ip + ":" + port
    log.Fatal(http.ListenAndServe(ip_port, nil))
}
