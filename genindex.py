#!/bin/python3
# -*- coding: utf-8 -*-

import re


lst = list()


with open("temp.file", "r") as f:
    lines = f.readlines()
    for line in lines:
        s = re.search("\\./.*?\\.md$", line)
        if s:
            path = s.group()
            filename = path.split("/")[-1]
            new_s = '<a href="%s">%s</a>' % (path, filename)
            line = line.replace(path, new_s)
        lst.append(line + '</br>')




head = '''
<!DOCTYPE html><script src="https://cdn.jsdelivr.net/npm/texme@0.9.0"></script>
<html>
<head>
<title>Page Title</title>
</head>
<body>

<form action="./query" method="POST">
    <div>
        <input name="search" id="search">
        <button>全文检索</button>
    </div>
    <hr>
</form>
'''

tail = '''
</body>
</html>
'''

f2 = open("index.html", "w")
f2.write(head)
f2.writelines(lst)
f2.write(tail)
f2.close()
