# mssws
most simple static web server

非常简单易用的静态 web 服务器，使用该工具，可以在 5 分钟内搭建出一个博客站点，不用对 Markdown 文件做任何修改即可实现不错的渲染效果



## 原理

使用 texme 渲染 markdown，highlight.js 对代码进行高亮，Go 编写简单的 Web 服务器以及将 markdown 转换成 texme 能识别的 html 文档



## 特点

可以十分简单的部署博客站点，并实现 Markown 和  LaTex 的渲染



##  使用

1. 将 markdown 文件放到 `main.go` 目录下，执行 `bash genindex.sh`
2.  执行 `go build main.go && ./main`



## 效果展示

genindex.sh 生成的导航页

![image-20201215012507495](image/image-20201215012507495.png)



genindex.sh 生成的导航页



![image-20201215012608805](image/image-20201215012608805.png)



md  文件渲染效果

![image-20201215012839972](image/image-20201215012839972.png)

md  文件渲染效果

![image-20201215012905597](image/image-20201215012905597.png)



## 使用的工具或者库

- texme
- highlight.js
