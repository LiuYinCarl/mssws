#!/bin/bash

# tree with directories
# tree ./blog -f -N -P "*.md|*.pdf" -I "*.assets" --prune --ignore-case > temp.file

# tree without directories
tree ./blog -i -f -N -P "*.md|*.pdf" -I "*.assets" --prune --ignore-case | grep "pdf$\|md$" > temp.file


if hash python3 2>/dev/null; then
    python3 genindex.py
else
    python genindex.py
fi

if [ $? -ne 0 ]; then
    echo "run python failed"
else
    echo "run python succ"
fi

rm temp.file

# 输出当前目录到 query.data 文件，用来实现查询功能
# pdf 文件不可进行全文查找
find ./blog -type f -name "*.md" > query.data
