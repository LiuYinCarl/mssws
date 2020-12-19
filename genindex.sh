#!/bin/bash

tree ./blog -f -P "*.md" -I "*.assets" > temp.file

python genindex.py

rm temp.file

# 输出当前目录到 query.data 文件，用来实现查询功能
find ./blog -type f -name "*.md" > query.data
