#!/bin/bash

tree ./blog -f -P "*.md|*.pdf" -I "*.assets" --prune --ignore-case > temp.file


if hash python3 2>/dev/null; then
	python3 genindex.py
else
	python genindex.py
fi

rm temp.file

# 输出当前目录到 query.data 文件，用来实现查询功能
# pdf 文件不可进行全文查找
find ./blog -type f -name "*.md" > query.data
