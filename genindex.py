#!/bin/python3
# -*- coding: utf-8 -*-

import re
import os
import json
from urllib.request import quote, unquote

def read_conf():
    with open("./config.json", "r") as f:
        conf = json.load(f)
        return conf

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

def gen_rss_file(items):
    conf = read_conf()

    head = """<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
  <title>{}</title>
  <link>{}</link>
  <atom:link href="{}" rel="self" type="application/rss+xml" />
  <description>{}</description>
  <language>en-us</language>
""".format(conf["RssTitle"], conf["RssLink"], conf["RssLink"], conf["RssDesc"])

    tail = """
</channel>
</rss>
"""

    rss_items = []
    for item in items:
        title = item[1]
        link = "{}{}".format(conf["SiteLink"], item[0].lstrip("."))
        link = quote(link, safe=";/?:@&=+$,", encoding="utf-8")
        desc = title
        s = """
  <item>
    <title>{}</title>
    <link>{}</link>
    <description>{}</description>
    <guid>{}</guid>
  </item>
""".format(title, link, desc, link)
        rss_items.append(s)

    item_info = "".join(rss_items)

    with open(conf["RssFile"], "w") as f:
        f.write(head + item_info + tail)

def gen_index_file(index_file_line):
    with open("index.data", "w") as f2:
        f2.writelines(index_file_line)


index_file_line = []
rss_items = []

with open("temp.file", "r") as f:
    lines = f.readlines()
    for line in lines:
        s = re.search("\\./.*?\\.md$", line, re.I)
        if s:
            path = s.group()
            filename = path.split("/")[-1]
            filename_check(filename)
            new_s = '<a href="{}">{}</a>'.format(path, filename)
            rss_items.append((path, filename))
            line = line.replace(path, new_s)
            index_file_line.append(line + '</br>')
            continue

        s = re.search("\\./.*?\\.pdf$", line, re.I)
        if s:
            path = s.group()
            filename = path.split("/")[-1]
            filename_check(filename)
            new_s = '<a href="./lib/pdfjs/web/viewer.html?file=/{}">{}</a>'.format(path, filename)
            rss_items.append((path, filename))
            line = line.replace(path, new_s)
            index_file_line.append(line + '</br>')
            continue
        index_file_line.append(line + '</br>')

gen_index_file(index_file_line)
gen_rss_file(rss_items)

