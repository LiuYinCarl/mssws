#!/bin/python3
# -*- coding: utf-8 -*-

import re
import os

def filename_check(filename):
    if " " in filename:
        print("[file name check] {}: space in filename".format(filename))
        return False
        
    suffix = os.path.splitext(filename)[-1] 
    if suffix.lower() == ".pdf" and suffix != ".pdf":
        print("[file name check] {} : file suffix error, use '.pdf'".format(filename))
        return False
        
    if suffix.lower() == ".md" and suffix != ".md":
        print("[file name check] {} : file suffix error, use '.md'".format(filename))
        return False

    return True


lst = list()


with open("temp.file", "r") as f:
    lines = f.readlines()
    for line in lines:
        s = re.search("\\./.*?\\.md$", line, re.I)
        if s:
            path = s.group()
            filename = path.split("/")[-1]
            filename_check(filename)
            new_s = '<a href="%s">%s</a>' % (path, filename)
            line = line.replace(path, new_s)
            lst.append(line + '</br>')
            continue
        
        s = re.search("\\./.*?\\.pdf$", line, re.I)
        if s:
            path = s.group()
            filename = path.split("/")[-1]
            filename_check(filename)
            new_s = '<a href="./pdfjs-2.7.570-dist/web/viewer.html?file=/%s">%s</a>' %(path, filename)
            line = line.replace(path, new_s)
            lst.append(line + '</br>')
            continue
        lst.append(line + '</br>')



with open("index.data", "w") as f2:
    f2.writelines(lst)
