#!/usr/bin/env python3
"""
八字古籍 TXT → Markdown 转换脚本
处理 PDF 提取文本的常见问题：
1. 字符间多余空格
2. 断行拼接
3. 页码/页眉页脚清理
4. 编码检测与统一
5. 章节结构提取
"""

import os, re, sys

SRC = "/Users/wikiglobal/workSapce/suanming-agent/docs/reference/bazi-classics"
DST = "/Users/wikiglobal/workSapce/suanming-agent/docs/reference/bazi-classics-md"

# ─── 工具函数 ───────────────────────────────────────────

def detect_and_read(filepath):
    """尝试多种编码读取文件"""
    for enc in ['utf-8', 'gbk', 'gb2312', 'gb18030', 'utf-16', 'big5', 'latin-1']:
        try:
            with open(filepath, 'r', encoding=enc) as f:
                content = f.read()
            if len(content) > 100:
                return content, enc
        except:
            continue
    return None, None


def clean_text(text: str) -> str:
    """清洗 PDF 提取产生的伪影"""
    # 1. 移除 PDF 常见页眉页脚
    text = re.sub(r'[""][^""]{1,30}提供下载[^""]*[""]', '', text)
    text = re.sub(r'http://[^\s]*', '', text)
    text = re.sub(r'易讯网.*?\n', '\n', text)
    text = re.sub(r'许昌知砚斋整理', '', text)

    # 2. 中文之间多余空格：中中 间 → 中间
    text = re.sub(r'(?<=[一-鿿])\s+(?=[一-鿿])', '', text)

    # 3. 中文和标点之间的空格
    text = re.sub(r'(?<=[一-鿿])\s+(?=[，。、；：！？])', '', text)
    text = re.sub(r'(?<=[，。、；：！？])\s+(?=[一-鿿])', '', text)

    # 4. 连续空行压缩为最多2个
    text = re.sub(r'\n{4,}', '\n\n\n', text)

    # 5. 行首行尾空白
    lines = [line.strip() for line in text.split('\n')]

    # 6. 智能合并：仅合并明显被PDF切断的短行
    merged = []
    i = 0
    while i < len(lines):
        line = lines[i]
        if not line:
            merged.append('')
            i += 1
            continue

        # 判断是否为章节标题（不要合并）
        is_title = bool(re.match(r'^[第卷]?[一二三四五六七八九十百千\d]+[、．.\s]', line))

        # 如果是短行且不是标题，尝试与下一行合并
        if not is_title and len(line) < 25 and i + 1 < len(lines):
            next_line = lines[i + 1]
            if next_line and re.match(r'^[一-鿿]', next_line):
                merged.append(line + next_line)
                i += 2
                continue

        # 如果当前行以非句号结尾，且下一行以中文开头，合并（限定在中等长度）
        if not is_title and i + 1 < len(lines):
            next_line = lines[i + 1]
            if (next_line and len(line) > 5 and not line[-1] in '。！？"」』）'):
                if re.match(r'^[一-鿿（《「]', next_line):
                    merged.append(line + next_line)
                    i += 2
                    continue

        merged.append(line)
        i += 1

    return '\n'.join(merged)


def add_markdown_structure(text: str, source_type: str) -> str:
    """根据文件类型添加 Markdown 格式"""
    lines = text.split('\n')
    result = []
    title = lines[0] if lines else ''

    # 添加文档标题
    if title and len(title) < 50 and not title.startswith('#'):
        result.append(f'# {title}')
        result.append('')
        lines = lines[1:]  # remove deduplicated title line

    for i, line in enumerate(lines):
        stripped = line.strip()
        if not stripped:
            result.append('')
            continue

        # 跳过已经有的书名标记
        if stripped in title or stripped == title:
            continue

        # 检测章节标题模式
        # 中国数字标题：一、二、三... 或 卷一、卷二...
        if re.match(r'^[一二三四五六七八九十百千]+[、．.\s]', stripped):
            result.append(f'\n## {stripped}')
        elif re.match(r'^卷[一二三四五六七八九十百千]+', stripped):
            result.append(f'\n## {stripped}')
        # 第X章/篇/节
        elif re.match(r'^第[一二三四五六七八九十百千\d]+[章节篇]', stripped):
            result.append(f'\n### {stripped}')
        # 通神论/六亲论 等大章节
        elif stripped in ['通神论', '六亲论'] or re.match(r'^[上中下]篇', stripped):
            result.append(f'\n# {stripped}')
        # 十天干标题
        elif re.match(r'^[甲乙丙丁戊己庚辛壬癸]、', stripped):
            result.append(f'**{stripped}**')
        else:
            result.append(stripped)

    return '\n'.join(result)


def convert_file(filename, source_type):
    """主转换函数"""
    src_path = os.path.join(SRC, filename)
    dst_path = os.path.join(DST, filename.replace('.txt', '.md'))

    print(f"📄 处理: {filename}")

    text, enc = detect_and_read(src_path)
    if not text:
        print(f"   ❌ 无法读取")
        return None

    print(f"   编码: {enc}, 原始: {len(text)} 字符")

    # 清洗
    text = clean_text(text)
    print(f"   清洗后: {len(text)} 字符")

    # 添加结构
    text = add_markdown_structure(text, source_type)

    # 保存
    os.makedirs(DST, exist_ok=True)
    with open(dst_path, 'w', encoding='utf-8') as f:
        f.write(text)

    size_kb = os.path.getsize(dst_path) / 1024
    lines = len(text.split('\n'))
    print(f"   ✅ 保存: {dst_path} ({size_kb:.0f}KB, {lines}行)")
    return dst_path


def convert_yuanhai():
    """转换渊海子平"""
    src = "/Users/wikiglobal/workSapce/suanming-agent/docs/reference/yuanhai-ziping-full.txt"
    dst = "/Users/wikiglobal/workSapce/suanming-agent/docs/reference/bazi-classics-md/渊海子平.md"

    text, enc = detect_and_read(src)
    if not text:
        print(f"❌ 渊海子平读取失败")
        return

    print(f"📄 处理: 渊海子平 ({enc}, {len(text)} 字符)")
    text = clean_text(text)
    text = add_markdown_structure(text, 'classic')

    os.makedirs(DST, exist_ok=True)
    with open(dst, 'w', encoding='utf-8') as f:
        f.write(text)
    print(f"   ✅ 保存: {dst} ({os.path.getsize(dst)/1024:.0f}KB)")


# ─── 主流程 ─────────────────────────────────────────────

if __name__ == '__main__':
    convert_file('ditiansui-extracted.txt', 'classic')
    convert_file('ziping-zhenquan-extracted.txt', 'classic')
    convert_file('qiongtong-baojian-extracted.txt', 'classic')
    convert_file('八字---三命通会.txt', 'classic')
    convert_file('八字---格局论命.txt', 'modern')
    convert_yuanhai()

    # 统计
    print("\n" + "="*60)
    total = 0
    for f in sorted(os.listdir(DST)):
        size = os.path.getsize(os.path.join(DST, f))
        total += size
        print(f"  {f}: {size/1024:.0f}KB")
    print(f"  合计: {total/1024:.0f}KB")
