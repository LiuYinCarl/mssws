#!/bin/python

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
<!DOCTYPE html>
<html>
<head>
<title>Page Title</title>
</head>
<body>
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
