# 命理大师前端重设计

> 状态：已批准 | 日期：2026-06-15 | 风格：亮底扁平插画卡片（Modern Tarot方向）

## 设计方向

**亮底 + 几何插画 + 卡片布局**。参考 Modern Witch 塔罗牌的设计语言：奶油底色、细线描边、克制的薄涂色彩、仪式感来自排版和几何符号。

### 对比参考

| 元素 | v1（暗黑） | v2（彩色卡通） | v3（线稿暗底） | **v4（最终）** |
|------|-----------|---------------|---------------|---------------|
| 底色 | #0D0C0A | #f7f5f2 | #1c1b1a | **#f2efe9** |
| 插画 | emoji | 填充卡通+笑脸 | 纯线稿 | **几何薄涂+细描边** |
| 字体 | Inter | Playfair+Inter | Playfair+Inter | **Playfair+Inter** |
| 氛围 | 暗黑 | 可爱 | 极暗神秘 | **明亮+柔和神秘** |

---

## 1. 设计 Token

### 配色

```css
--bg:            #f2efe9;   /* 暖奶油底 */
--bg-secondary:  #faf8f5;   /* 卡片白 */
--bg-hover:      #f5f1ea;   /* 悬停 */
--border:        #e4ded4;   /* 暖灰边框 */
--border-light:  #ede8df;   /* 极淡边框 */
--text-primary:  #3a3632;   /* 炭灰正文 */
--text-secondary:#6b6050;   /* 中灰 */
--text-muted:    #8a7a68;   /* 淡灰 */
--accent:        #c4a978;   /* 暖金强调 */
--accent-bg:     rgba(184,149,106,0.1);
--accent-dim:    #8a7a60;   /* 暗金 */

/* 五行功能色 */
--wx-wood:   #7a9e7e;
--wx-fire:   #c47a6a;
--wx-earth:  #b8956a;
--wx-metal:  #c4a96a;
--wx-water:  #6b8aa8;
```

### 字体

```css
--serif: "Playfair Display", serif;  /* 标题/干支大字 */
--sans:  "Inter", system-ui, sans-serif;
--mono:  "SF Mono", "JetBrains Mono", monospace;

font-size: 15px;
line-height: 1.65;
```

### 圆角与阴影

```css
--radius-sm: 6px;  --radius-md: 10px;  --radius-lg: 14px;
--shadow-card: 0 8px 30px rgba(58,54,50,0.08);
--shadow-glow: 0 0 0 4px rgba(184,149,106,0.1);
```

---

## 2. 图标体系

### 五行精灵（5个 — 核心视觉资产，手绘 SVG）

| 元素 | 形态 | 笔法 |
|------|------|------|
| 木 | 圆冠树 + 细干 + 叶片 | 薄涂绿 + 细描边 |
| 火 | 水滴形焰体 + 火尖 + 小火簇 | 薄涂暖橙 + 细描边 |
| 土 | 叠三角山 + 小晶块 | 薄涂暖褐 + 细描边 |
| 金 | 菱形叠套 + 放射十字线 | 薄涂暖金 + 细描边 |
| 水 | 椭圆水滴 + 波纹弧 | 薄涂雾蓝 + 细描边 |

40-56px 尺寸，无面部，纯几何。

### 九星（9个 — 模板化生成）

统一圆形底模板，每星改 3 个参数：中心几何形状、芒数/线型、背景纹底色。

### 八门（8个 — 模板化生成）

统一方形底模板，抽象门符号：线条开合走向区分。

### 八神（8个）

不需要图标。文字标签 + 色标即可。

### 其他

- 八卦：Unicode ☰☱☲☳☴☵☶☷
- 干支：本身即视觉元素
- UI图标：lucide-vue-next

---

## 3. 布局

### 空状态
- 垂直居中，暖奶油底
- 品牌名 Playfair Display 28px
- 3列快捷提问卡片（暗金线框图标 + 标签）
- 输入框 52px高，max-width 520px，focus 暗金光晕

### 聊天状态
- 消息区中间可滚动
- 输入栏固定底部
- 空→聊过渡：0.35s cubic-bezier

---

## 4. 组件改造

### 八字命盘
- 4列暗金卡片，每张：宫位标签 + 精灵插画(48px) + 干支大字(Playfair 28px) + 十神
- 自绘详情表（暖底表头 + 淡格线）
- 自绘五行条（6px高）+ 大运chip
- 四柱 stagger 展开动画

### 奇门遁甲盘
- 3x3 卡片牌阵
- 每格：宫位名 + 五行插画(26px) + 星+门+神文字 + 干支
- 值符宫暗金边框 + 光晕
- 九宫中→外 stagger 显现

### 其他组件
- AssistantTurn：竖线时间线链路 + accent跳动点加载
- ChatPanel：暗底输入框 + 图标发送按钮
- WelcomePanel：线稿图标 + 3列卡片

---

## 5. 技术实现

- Vue 3 + Naive UI + TypeScript
- 新增：lucide-vue-next
- 五行精灵：5个独立 Vue SVG 组件（`SpriteWood.vue`等）
- 九星/八门：参数化 SVG 组件（`SpriteStar.vue`、`SpriteDoor.vue`）
- 动画：纯 CSS
- 字体：Google Fonts 加载 Playfair Display

## 6. 变更文件

| 文件 | 改动 |
|------|------|
| `web/index.html` | +Google Fonts |
| `web/package.json` | +lucide-vue-next |
| `web/src/style.css` | 全局Token重写 |
| `web/src/components/sprites/*.vue` | **新增**：5个精灵+2个参数化组件 |
| `web/src/components/ChatPanel.vue` | 空状态居中，暗底输入框 |
| `web/src/components/WelcomePanel.vue` | 线稿图标，3列卡片 |
| `web/src/components/AssistantTurn.vue` | 竖线链路，统一对齐 |
| `web/src/components/BaziChartCard.vue` | 完全重写 |
| `web/src/components/QimenChart.vue` | 完全重写 |
| `web/src/components/ChatBubble.vue` | 适配新配色 |
