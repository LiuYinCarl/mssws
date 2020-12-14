#!/bin/bash

tree -f -P "*.md" -I "*.assets" > temp.file

python genindex.py
