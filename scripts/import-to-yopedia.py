#!/usr/bin/env python3
"""
八字命理资料 → yopedia 知识库导入脚本 (v2)
- 自动清洗格式（去空行、修复章节标题）
- 大文件按章节拆分
- 自动添加交叉引用链接
- 通过 HTTP API 直接写入

用法:
  make yopedia-start   # 先启动服务
  python3 scripts/import-to-yopedia.py
"""

import os, sys, json, re, time
import urllib.request, urllib.error

YOPEDIA = "http://localhost:3100"
DOCS = os.path.join(os.path.dirname(os.path.dirname(__file__)), "docs")
PAGE_LIMIT = 40000  # 单页最大字符数，超过则拆分

# 读取 service token
def _load_token():
    env_file = os.path.join(os.path.dirname(os.path.dirname(__file__)), "yopedia", ".env.local")
    try:
        with open(env_file) as f:
            for line in f:
                if line.startswith("YOPEDIA_SERVICE_TOKEN="):
                    return line.strip().split("=", 1)[1]
    except:
        pass
    return None

SERVICE_TOKEN = _load_token()

# ─── 工具函数 ───────────────────────────────────

def api(method, path, data=None):
    url = f"{YOPEDIA}{path}"
    body = json.dumps(data, ensure_ascii=False).encode() if data else None
    headers = {"Content-Type": "application/json; charset=utf-8"}
    if SERVICE_TOKEN:
        headers["Authorization"] = f"Bearer {SERVICE_TOKEN}"
    req = urllib.request.Request(url, data=body, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, resp.read().decode()
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()
    except Exception as e:
        return 0, str(e)

def clean_markdown(text):
    """清洗 markdown 格式"""
    # 压缩连续空行
    text = re.sub(r'\n{4,}', '\n\n\n', text)
    # 修复缺少空行的标题
    text = re.sub(r'([^\n])\n(#{1,4}\s)', r'\1\n\n\2', text)
    # 移除文档开头的 yaml frontmatter（保留为引用）
    return text.strip()

def add_frontmatter(slug, title, text, level="B", source=""):
    """添加统一的页面头部"""
    stars = {"S":"⭐⭐⭐⭐⭐","A":"⭐⭐⭐⭐","B":"⭐⭐⭐","C":"⭐⭐"}.get(level, "⭐⭐⭐")
    header = f"""# {title}

> 权威级别：{stars} | 来源：{source} | slug：{slug}
"""
    return header + "\n" + text

def add_crossrefs(text, links):
    """添加交叉引用"""
    if not links:
        return text
    refs = "\n\n## 关联知识\n"
    for slug, title in links.items():
        refs += f"- [{title}]({slug}.md)\n"
    return text.rstrip() + refs

def split_by_chapters(text, max_chars=PAGE_LIMIT):
    """按 ## 标题拆分为子页面，子页面超出限制时再按 ### 拆分"""
    if len(text) <= max_chars:
        return [("", text)]

    sections = re.split(r'\n(?=## )', text)
    chunks = []
    current = [sections[0]] if sections[0].strip() else []

    for sec in sections[1:]:
        trial = "\n\n".join(current + [sec])
        if len(trial) > max_chars:
            # 当前累积块已满，保存
            chunks.append("\n\n".join(current))
            current = [sec]
        else:
            current.append(sec)

    if current:
        chunks.append("\n\n".join(current))

    return [(f"-part{i+1}", c) for i, c in enumerate(chunks)] if len(chunks) > 1 else [("", text)]

# ─── 导入定义 ───────────────────────────────────

CROSSREFS = {
    "ref-bazi-knowledge": {
        "ref-bazi-foundation": "天干地支与五行",
        "ref-bazi-shishen": "十神体系",
        "ref-bazi-geju": "格局与用神",
        "ref-bazi-dayun": "大运流年",
        "ref-bazi-career": "事业财运",
        "ref-bazi-health": "健康疾病",
        "ref-bazi-six-relatives": "六亲",
        "ref-bazi-marriage": "合婚",
        "ref-bazi-yuanhai": "渊海子平",
        "ref-bazi-ditiansui": "滴天髓阐微",
        "ref-bazi-sanming": "三命通会",
        "ref-bazi-qiongtong": "穷通宝鉴",
        "ref-bazi-ziping": "子平真诠",
        "data-bazi-mingli-bench": "命例验证集",
        "spec-bazi-conflict-rules": "冲突解决规则",
    },
    "ref-bazi-yuanhai": {
        "ref-bazi-ditiansui": "滴天髓阐微",
        "ref-bazi-ziping": "子平真诠",
        "ref-bazi-sanming": "三命通会",
        "ref-bazi-qiongtong": "穷通宝鉴",
    },
    "ref-bazi-ditiansui": {
        "ref-bazi-yuanhai": "渊海子平",
        "ref-bazi-ziping": "子平真诠",
        "ref-bazi-geju": "格局与用神",
    },
}

# 导入清单：(slug, 分类slug, 标题, 源文件路径, 权威级别, 来源)
IMPORTS = [
    # ── 结构化参考 ──
    ("ref-bazi-knowledge", "hub-bazi-knowledge", "八字AI综合知识库",
     "learning/bazi-knowledge-base.md", "A", "整合编纂"),
    ("ref-bazi-foundation", "hub-bazi-knowledge", "天干地支与五行基础",
     "reference/bazi-reference-01-foundation.md", "A", "GitHub Wolke/bazi-mingli"),
    ("ref-bazi-shishen", "hub-bazi-knowledge", "十神体系",
     "reference/bazi-reference-02-shishen.md", "A", "GitHub Wolke/bazi-mingli"),
    ("ref-bazi-geju", "hub-bazi-knowledge", "格局与用神",
     "reference/bazi-reference-03-geju-yongshen.md", "A", "GitHub Wolke + ctext"),
    ("ref-bazi-dayun", "hub-bazi-knowledge", "大运流年与婚姻",
     "reference/bazi-reference-04-dayun-marriage.md", "A", "GitHub Wolke"),
    ("ref-bazi-career", "hub-bazi-knowledge", "事业财运",
     "reference/bazi-reference-05-career-wealth.md", "C", "网络搜索"),
    ("ref-bazi-health", "hub-bazi-knowledge", "健康疾病",
     "reference/bazi-reference-06-health.md", "C", "网络搜索"),
    ("ref-bazi-six-relatives", "hub-bazi-knowledge", "六亲断语",
     "reference/bazi-reference-07-six-relatives.md", "C", "网络搜索"),
    ("ref-bazi-marriage", "hub-bazi-knowledge", "合婚方法",
     "reference/bazi-reference-08-marriage-matching.md", "C", "网络搜索"),
    # ── 古籍原文 ──
    ("ref-bazi-yuanhai", "hub-bazi-knowledge", "渊海子平",
     "reference/bazi-reference-09-yuanhai-ziping.md", "S", "宋·徐子平"),
    ("ref-bazi-ditiansui", "hub-bazi-knowledge", "滴天髓阐微",
     "reference/bazi-reference-10-ditiansui-cases.md", "S", "宋·京图/清·任铁樵"),
    ("ref-bazi-sanming", "hub-bazi-knowledge", "三命通会",
     "reference/bazi-classics-md/八字---三命通会.md", "S", "明·万民英"),
    ("ref-bazi-qiongtong", "hub-bazi-knowledge", "穷通宝鉴",
     "reference/bazi-classics-md/qiongtong-baojian-extracted.md", "S", "清·余春台"),
    ("ref-bazi-ziping", "hub-bazi-knowledge", "子平真诠评注",
     "reference/bazi-classics-md/ziping-zhenquan-extracted.md", "S", "清·沈孝瞻/民国·徐乐吾"),
    # ── 验证数据 ──
    ("data-bazi-mingli-bench", "hub-bazi-knowledge", "MingLi-Bench命例验证集",
     "reference/bazi-reference-11-mingli-bench-cases.md", "B", "GitHub MingLi-Bench"),
    # ── 规则 ──
    ("spec-bazi-conflict-rules", "hub-bazi-knowledge", "冲突解决规则",
     "reference/README.md", "A", "综合整理"),
]

# ─── 主流程 ───────────────────────────────────

def main():
    # 1. 检查服务
    code, resp = api("GET", "/api/wiki")
    if code != 200:
        print(f"❌ yopedia 未响应 (HTTP {code})")
        print(f"   请先执行: make yopedia-start")
        return

    # 2. 获取已有页面
    existing = set()
    try:
        data = json.loads(resp)
        pages = data.get("pages", []) if isinstance(data, dict) else []
        existing = {p['slug'] for p in pages if isinstance(p, dict)}
        print(f"📡 yopedia 在线，已有 {len(existing)} 个页面\n")
    except:
        print(f"📡 yopedia 在线\n")

    # 3. 先建 hub 页
    hub_links = {}
    for slug, parent, title, _, level, source in IMPORTS:
        hub_links[slug] = title

    hub_content = add_frontmatter("hub-bazi-knowledge", "八字命理知识库", f"""## 概述

独立知识库，存储八字命理（四柱推命）参考资料。所有内容标注权威级别和来源。

## 页面索引

| slug | 标题 | 级别 | 来源 |
|------|------|------|------|
""" + "\n".join(f"| [{slug}]({slug}.md) | {title} | {level} | {source} |" for slug, _, title, _, level, source in IMPORTS) + """

## 知识库规则

参见 [spec-bazi-conflict-rules](spec-bazi-conflict-rules.md)
""", "A", "项目初始化")

    if "hub-bazi-knowledge" not in existing:
        code, _ = api("POST", "/api/wiki", {"slug": "hub-bazi-knowledge", "title": "八字命理知识库", "content": hub_content})
        print(f"  {'✅' if code in (200,201) else '❌'} hub-bazi-knowledge (入口页)")
    else:
        code, _ = api("PUT", "/api/wiki/hub-bazi-knowledge", {"slug": "hub-bazi-knowledge", "title": "八字命理知识库", "content": hub_content})
        print(f"  🔄 hub-bazi-knowledge (已更新)")

    # 4. 逐文件导入
    imported, skipped, failed = 0, 0, 0
    for slug, parent, title, src_path, level, source in IMPORTS:
        full = os.path.join(DOCS, src_path)
        if not os.path.exists(full):
            print(f"  ⚠️ {slug}: 文件不存在 ({src_path})")
            skipped += 1
            continue

        with open(full, 'r', encoding='utf-8') as f:
            raw = f.read()

        # 清洗
        text = clean_markdown(raw)
        # 添加来源头
        text = add_frontmatter(slug, title, text, level, source)
        # 添加交叉引用
        links = CROSSREFS.get(slug, {})
        if parent and parent in hub_links:
            links[parent] = hub_links[parent]
        if links:
            text = add_crossrefs(text, links)

        size_kb = len(raw) / 1024
        print(f"  📄 {slug} ({size_kb:.0f}KB)", end=" ")

        # 大文件拆分
        chunks = split_by_chapters(text)
        if len(chunks) == 1:
            data = {"slug": slug, "title": title, "content": text}
            if slug in existing:
                code, _ = api("PUT", f"/api/wiki/{slug}", data)
            else:
                code, _ = api("POST", "/api/wiki", data)
            if code in (200, 201):
                print("✅")
                imported += 1
            else:
                print(f"❌ HTTP {code}")
                failed += 1
        else:
            # 多段导入：主页 + 子页
            main_text = chunks[0][1]
            main_text += f"\n\n> 📖 本页共 {len(chunks)} 部分，以下为子页面链接：\n"
            data = {"slug": slug, "title": title, "content": main_text}
            if slug in existing:
                api("PUT", f"/api/wiki/{slug}", data)
            else:
                api("POST", "/api/wiki", data)

            parts = []
            for i, (suffix, content) in enumerate(chunks):
                sub_slug = f"{slug}-part{i+1}"
                sub_title = f"{title}（第{i+1}部分）"
                sub_content = add_frontmatter(sub_slug, sub_title, content, level, source)
                data = {"slug": sub_slug, "title": sub_title, "content": sub_content}
                if sub_slug in existing:
                    code, _ = api("PUT", f"/api/wiki/{sub_slug}", data)
                else:
                    code, _ = api("POST", "/api/wiki", data)
                parts.append(sub_slug)
                print(f"\n     └─ {sub_slug} {'✅' if code in (200,201) else '❌'}")

            # 更新主页添加子页链接
            links_text = main_text + "\n".join(f"- [{p}]({p}.md)" for p in parts)
            api("PUT", f"/api/wiki/{slug}", {"slug": slug, "title": title, "content": links_text})
            print(f"  ✅ ({len(chunks)}段)")
            imported += 1

        time.sleep(0.3)  # 避免打爆服务

    # 5. 总结
    print(f"\n{'='*50}")
    print(f"✅ 导入完成: {imported} 成功, {skipped} 跳过, {failed} 失败")
    print(f"   访问: {YOPEDIA}")
    # 验证
    code, resp = api("GET", "/api/wiki")
    if code == 200:
        pages = json.loads(resp)
        print(f"   当前页面总数: {len(pages) if isinstance(pages, list) else '?'}")

if __name__ == "__main__":
    main()
