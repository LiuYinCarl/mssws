#!/bin/bash

if command -v git >/dev/null 2>&1; then
    echo "git installed, start update libs..."
else
    echo "git not install, exit..."
    exit
fi

backup_dir="backup"

mkdir "${backup_dir}"

mv ./mathjax "${backup_dir}"
mv ./marked  "${backup_dir}"
mv ./texme   "${backup_dir}"

git clone --depth=1 https://github.com/susam/texme.git
git clone --depth=1 https://github.com/markedjs/marked.git
git clone --depth=1 https://github.com/mathjax/mathjax.git

cd ./mathjax || exit
rm -rf .git
cd ..

cd ./marked || exit
rm -rf .git
cd ..

cd texme || exit
rm -rf .git
cd ..

rm -rf "${backup_dir}"
