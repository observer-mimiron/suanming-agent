#!/usr/bin/env python3
"""
智能导入 v3 — 低成本方案
1. 机械按 ## 标题拆分（无 LLM）
2. 批量 LLM 生成 tags + slug（一次调用处理所有章节，用 flash 模型）
3. 创建 wiki 页面
"""

import os, json, re, time, urllib.request, urllib.error

KNOWLEDGE_URL = "http://localhost:3100"
DOCS = os.path.join(os.path.dirname(os.path.dirname(__file__)), "docs")
LLM_KEY = "sk-d067569ff9424966aa190534eacfb75a"
LLM_URL = "https://api.deepseek.com/chat/completions"
LLM_MODEL = "deepseek-chat"  # flash 模型更便宜
CHUNK_MAX = 1500

def load_token():
    env = os.path.join(os.path.dirname(os.path.dirname(__file__)), "knowledge", ".env.local")
    try:
        with open(env) as f:
            for line in f:
                if "SERVICE_TOKEN=" in line:
                    return line.strip().split("=",1)[1]
    except: pass

TOKEN = load_token()

def api(method, path, data=None):
    url = f"{KNOWLEDGE_URL}{path}"
    body = json.dumps(data, ensure_ascii=False).encode() if data else None
    h = {"Content-Type": "application/json; charset=utf-8"}
    if TOKEN: h["Authorization"] = f"Bearer {TOKEN}"
    req = urllib.request.Request(url, body, h, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as r:
            return r.status, json.loads(r.read())
    except urllib.error.HTTPError as e:
        return e.code, json.loads(e.read())
    except Exception as e:
        return 0, {"error": str(e)}

def llm_tag(sections_info):
    """批量生成 tags 和 slug，一次 LLM 调用"""
    prompt = f"""为以下 {len(sections_info)} 个章节生成 slug 和 tags。

规则：
- slug: 英文短标识 (kebab-case, ≤20字符)
- tags: 3-5个中文标签

返回纯 JSON 数组:
[{{"idx":0,"slug":"xxx","tags":["标签1","标签2"]}},...]
只返回 JSON。"""

    content = "\n".join(f"{i}. {s}" for i, s in enumerate(sections_info))
    data = {
        "model": LLM_MODEL,
        "messages": [{"role":"system","content":prompt},{"role":"user","content":content}],
        "temperature": 0.1, "max_tokens": 2000,
    }
    req = urllib.request.Request(LLM_URL, json.dumps(data,ensure_ascii=False).encode(),
        {"Content-Type":"application/json","Authorization":f"Bearer {LLM_KEY}"})
    try:
        with urllib.request.urlopen(req, timeout=60) as r:
            result = json.loads(r.read())["choices"][0]["message"]["content"]
    except Exception as e:
        print(f"    ❌ LLM 失败: {e}")
        return None

    result = result.strip()
    if result.startswith("```"): result = re.sub(r'^```\w*\n?','',result); result = re.sub(r'\n?```$','',result)
    m = re.search(r'\[.*\]', result, re.DOTALL)
    if m:
        try: return json.loads(m.group())
        except: pass
    try: return json.loads(result)
    except: return None


def split_markdown(content):
    """按 ## 标题机械拆分，超长段落按 \\n\\n 继续拆"""
    sections = re.split(r'\n(?=## )', content)
    chunks = []

    for sec in sections:
        sec = sec.strip()
        if not sec: continue

        # 提取标题
        title_match = re.match(r'^##\s+(.+)', sec)
        title = title_match.group(1).strip() if title_match else sec[:30]

        # 去掉标题行本身得到正文
        body = re.sub(r'^##\s+.+\n', '', sec).strip() if title_match else sec

        if len(body) <= CHUNK_MAX:
            chunks.append({"title": title, "content": sec})
        else:
            # 按段落拆分
            paras = re.split(r'\n\n+', body)
            sub = []
            current = ""
            for p in paras:
                if len(current) + len(p) > CHUNK_MAX and current:
                    sub.append(current.strip())
                    current = p
                else:
                    current += ("\n\n" + p) if current else p
            if current: sub.append(current.strip())

            for j, s in enumerate(sub):
                suffix = f"（{j+1}）" if len(sub) > 1 else ""
                header = f"## {title}{suffix}\n\n" if title_match else ""
                chunks.append({"title": f"{title}{suffix}", "content": header + s})

    return chunks


def import_document(slug_prefix, title, filepath, level, source):
    """导入一个文档：拆分 → LLM打标签 → 创建页面"""
    fp = os.path.join(DOCS, filepath)
    if not os.path.exists(fp):
        print(f"  ⚠️ 文件不存在")
        return 0

    with open(fp, encoding='utf-8') as f:
        content = f.read()

    size_kb = len(content) / 1024
    if size_kb < 5:
        print(f"  {title} ({size_kb:.0f}KB) — 跳过")
        return 0

    # 机械拆分
    chunks = split_markdown(content)
    if len(chunks) <= 1:
        print(f"  {title} — 无需拆分 ({len(chunks)}块)")
        return 0

    print(f"  {title} ({size_kb:.0f}KB) → {len(chunks)} 块", end=" ", flush=True)

    # 批量 LLM 标签
    titles = [c["title"] for c in chunks]
    tags_result = llm_tag(titles)

    if tags_result:
        tag_map = {t.get("idx",i): t for i, t in enumerate(tags_result)}
        for i, chunk in enumerate(chunks):
            t = tag_map.get(i, {})
            chunk["slug"] = t.get("slug", f"s{i+1}")
            chunk["tags"] = t.get("tags", [])
        print("→ LLM已标注", end=" ", flush=True)
    else:
        for i, chunk in enumerate(chunks):
            chunk["slug"] = f"s{i+1}"
            chunk["tags"] = []
        print("→ 默认标注", end=" ", flush=True)

    # 删除旧页面
    code, resp = api("GET", "/api/wiki")
    existing = {p['slug'] for p in resp.get('pages',[])} if code==200 else set()
    for old in list(existing):
        if old.startswith(slug_prefix):
            api("DELETE", f"/api/wiki/{old}")

    # 创建子页面
    stars = {"S":"⭐⭐⭐⭐⭐","A":"⭐⭐⭐⭐","B":"⭐⭐⭐","C":"⭐⭐"}.get(level,"⭐⭐⭐")
    created = 0
    parent_links = []

    for i, ch in enumerate(chunks):
        if not ch["content"].strip(): continue
        full_slug = f"{slug_prefix}-{ch['slug']}"
        tags = (ch.get("tags",[]) or []) + ["八字", f"父:{slug_prefix}"]

        nav = ""
        if i > 0:
            nav += f"← [{chunks[i-1]['title']}]({slug_prefix}-{chunks[i-1]['slug']}.md)  "
        nav += f"[📑 目录]({slug_prefix}.md)"
        if i < len(chunks) - 1:
            nav += f"  → [{chunks[i+1]['title']}]({slug_prefix}-{chunks[i+1]['slug']}.md)"

        text = f"# {ch['title']}\n\n> 权威级别：{stars} | 来源：{source}\n\n{ch['content']}\n\n---\n{nav}"

        code, _ = api("POST", "/api/wiki", {"slug": full_slug, "content": text, "tags": tags})
        if code == 201: created += 1
        time.sleep(0.15)

    # 父索引页
    parent_content = f"# {title}\n\n> 权威级别：{stars} | 来源：{source}\n\n## 目录\n\n" + \
        "\n".join(f"- [{chunks[i]['title']}]({slug_prefix}-{chunks[i]['slug']}.md)" for i, ch in enumerate(chunks) if ch["content"].strip())
    api("POST", "/api/wiki", {"slug": slug_prefix, "content": parent_content,
        "tags": ["八字","古籍","目录",f"级别:{level}"]})

    print(f"→ {created} 页")
    return created


def main():
    code, resp = api("GET", "/api/wiki")
    if code != 200: print("❌ 未连接"); return
    print(f"📡 知识库 {len(resp.get('pages',[]))} 页\n")

    files = [
        ("ref-bazi-sanming", "三命通会", "reference/bazi-classics-md/八字---三命通会.md", "S", "明·万民英"),
        ("ref-bazi-ziping", "子平真诠评注", "reference/bazi-classics-md/ziping-zhenquan-extracted.md", "S", "清·沈孝瞻"),
        ("ref-bazi-ditiansui", "滴天髓阐微", "reference/bazi-classics-md/ditiansui-extracted.md", "S", "宋·京图/清·任铁樵"),
        ("ref-bazi-qiongtong", "穷通宝鉴", "reference/bazi-classics-md/qiongtong-baojian-extracted.md", "S", "清·余春台"),
        ("ref-bazi-yuanhai", "渊海子平", "reference/bazi-classics-md/渊海子平.md", "S", "宋·徐子平"),
        ("ref-bazi-gelulunming", "格局论命", "reference/bazi-classics-md/八字---格局论命.md", "A", "现代·黄大陆"),
    ]

    total = 0
    for prefix, title, path, level, src in files:
        total += import_document(prefix, title, path, level, src)
    print(f"\n✅ 共 {total} 页")

if __name__ == "__main__":
    main()
