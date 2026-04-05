#!/bin/bash

# tree with directories
# tree ./blog -f -N -P "*.md|*.pdf" -I "*.assets" --prune --ignore-case > temp.file

# tree without directories
tree ./blog -i -f -N -P "*.md" -I "*.assets" --prune --ignore-case | grep "pdf$\|md$" > temp.file
echo "<hr>" >> temp.file
tree ./blog -i -f -N -P "*.pdf" -I "*.assets" --prune --ignore-case | grep "pdf$\|md$" >> temp.file


# Require python3 and run the index generator
if ! command -v python3 >/dev/null 2>&1; then
    echo "Error: python3 not found. Please install Python 3."
    exit 1
fi

if ! python3 genindex.py; then
    echo "Error: python3 genindex.py failed."
    echo "You may need to install the toml module: pip3 install toml"
    exit 1
fi
echo "run python succ"

rm temp.file

# 输出当前目录到 query.data 文件，用来实现查询功能
# pdf 文件不可进行全文查找
find ./blog -type f -name "*.md" > query.data
