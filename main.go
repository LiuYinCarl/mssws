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
)


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

          return html + "<a href=\"http://47.93.196.173:7878/index.html\">Home Page</a>\n\n"
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


var is_head = true


func main() {
    http.HandleFunc("/", index)
    log.Fatal(http.ListenAndServe("0.0.0.0:7878", nil))
}

func index(w http.ResponseWriter, r *http.Request) {
    // w.Write([]byte(r.URL.Path))
    path := fmt.Sprintf(".%s", r.URL.Path)
    
    if ok := IsFile(path); !ok {
        w.Write([]byte("404 file not exist."))
        return
    }
    content := Md2html(path)
    w.Write(content)

    // content, _ := ioutil.ReadFile(path)
    // w.Write(content)
}
