package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	einomodel "github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/specialists"
	"github.com/wikiglobal/suanming-agent/internal/state"
	"github.com/wikiglobal/suanming-agent/internal/tools"
)

// specialistsConfig 是 specialists.Config 的类型别名，用于 route-bound 选择。
type specialistsConfig = specialists.Config

// AgentBuilder 负责从 specialist 配置构建 ADK ChatModelAgent 和 AgentTool。
// 持有共享的 ToolCallingChatModel 和工具 Registry。
type AgentBuilder struct {
	model          einomodel.ToolCallingChatModel
	reg            *tools.Registry
	llmModel       string
	flashChat      llm.Chat
	summarizerModel einomodel.ToolCallingChatModel
}

// NewAgentBuilder 创建 AgentBuilder。
// summarizerModel 用于 summarization 中间件压缩长对话历史，通常应配置为成本较低的 flash 模型。
func NewAgentBuilder(model einomodel.ToolCallingChatModel, reg *tools.Registry, flashChat llm.Chat, summarizerModel einomodel.ToolCallingChatModel) *AgentBuilder {
	return &AgentBuilder{model: model, reg: reg, flashChat: flashChat, summarizerModel: summarizerModel}
}

// SetLLMModel 设置用于追踪 span 的模型名称。
func (b *AgentBuilder) SetLLMModel(model string) { b.llmModel = model }

// noFStringGenModelInput 是跳过 FString 模板格式化的 GenModelInput。
// 当 instruction 包含字面花括号（如 JSON 数据块）时，用此函数替代默认的 defaultGenModelInput，
// 避免 Eino 将 {key} 当成 SessionValues 模板变量而报 NodeRunError。
func noFStringGenModelInput(ctx context.Context, instruction string, input *adk.AgentInput) ([]*schema.Message, error) {
	msgs := make([]*schema.Message, 0, len(input.Messages)+1)
	if instruction != "" {
		msgs = append(msgs, schema.SystemMessage(instruction))
	}
	msgs = append(msgs, input.Messages...)
	return msgs, nil
}

// BuildSpecialist 从 Config 构建一个领域专家 ChatModelAgent。
// 如果会话状态中已有出生资料或命盘结果，会被注入到 instruction 中。
func (b *AgentBuilder) BuildSpecialist(ctx context.Context, cfg specialists.Config, st *state.SessionState) (adk.Agent, error) {
	adapters, err := BuildAdaptersFor(b.reg, cfg.ToolNames, b.flashChat)
	if err != nil {
		return nil, err
	}
	// 会话已有上下文（含权威命盘数据）通过 instruction 中的 {{SESSION_CONTEXT}} 占位符注入，
	// 让 prompt 作者控制数据位置，避免 LLM 因长 instruction 注意力稀释而幻觉其他出生年份。
	// 占位符不存在时兜底拼到 instruction 最前面。
	var sessionCtx string
	if st != nil && (len(st.Profile) > 0 || st.HasBaziResult() || st.QimenResult != nil || st.HasZiWeiResult()) {
		sessionCtx = "## 会话已有上下文\n\n以下资料已在当前会话中提供，**直接使用，无需再次索要或调用工具获取**：\n"
		if len(st.Profile) > 0 {
			pb := NewBuilder() // 只用 buildProfileSection
			sessionCtx += "\n### 出生资料\n" + pb.buildProfileSection(st) + "\n"
		}
		if st.HasBaziResult() {
			sessionCtx += "\n### 命盘结果（已就绪，严禁重新调用排盘/用神/大运工具）\n"
			sessionCtx += b.buildBaziDataBlock(st)
		}
		if st.QimenResult != nil {
			sessionCtx += "\n### 奇门遁甲盘结果（已就绪，严禁重新调用 qimen_dunjia 工具）\n"
			sessionCtx += b.buildQimenDataBlock(st)
		}
		if st.HasZiWeiResult() {
			sessionCtx += "\n### 紫微命盘结果（已就绪，严禁重新调用 ziwei_calc；ziwei_liunian 仅在用户明确询问非当前年份时调用）\n"
			sessionCtx += b.buildZiWeiDataBlock(st)
		}
	}
	const sessionCtxPlaceholder = "{{SESSION_CONTEXT}}"
	instruction := cfg.Instruction
	if sessionCtx != "" {
		if strings.Contains(instruction, sessionCtxPlaceholder) {
			instruction = strings.Replace(instruction, sessionCtxPlaceholder, sessionCtx, 1)
		} else {
			instruction = sessionCtx + "\n\n---\n\n" + instruction
		}
	}
	instruction += "\n\n## 当前运行时上下文\n当前系统日期：" + time.Now().Format("2006-01-02") + "。\n当前系统时区：Asia/Shanghai。"
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   instruction,
		Model:         b.model,
		MaxIterations: 15,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: adapters,
			},
		},
		Handlers:         b.buildSpecialistHandlers(),
		GenModelInput:    noFStringGenModelInput,
		ModelRetryConfig: defaultRetryConfig(),
	})
}

// buildSpecialistHandlers 返回 specialist agent 的中间件链。
// 仅挂载 reduction（knowledge_search 截断）；不挂 summarization。
// 原因：summarization 的 Trigger 配置只设 ContextMessages=20、未设 ContextTokens，
// 导致 getTriggerContextTokens 返回 0，tokens>0 恒成立——每次 specialist 调用都触发
// summarizer 改写输入，污染注入的权威命盘数据（产生"好的我们继续"+日主幻觉+跳流年）。
// 会话历史压缩由 orchestrator.recordTurnAndMaintainContext + RunningSummary 负责。
func (b *AgentBuilder) buildSpecialistHandlers() []adk.ChatModelAgentMiddleware {
	var handlers []adk.ChatModelAgentMiddleware
	if mw, err := buildToolReductionMiddleware(); err == nil && mw != nil {
		handlers = append(handlers, mw)
	}
	return handlers
}

// buildSummarizationMiddleware 构造压缩长对话历史的中间件。
// 触发条件：对话消息数超过 20 条（约 10 轮 user/assistant 来回）。
// 摘要模型复用 AgentBuilder 的 flash 模型，避免用主模型做压缩推高成本。
// UserInstruction 约束摘要模型保留命理术语原文，防止"用神"/"十神"等术语被意译扭曲。
func buildSummarizationMiddleware(m einomodel.ToolCallingChatModel) (adk.ChatModelAgentMiddleware, error) {
	if m == nil {
		return nil, nil
	}
	return summarization.New(context.Background(), &summarization.Config{
		Model: m,
		Trigger: &summarization.TriggerCondition{
			ContextMessages: 20,
		},
		UserInstruction: "你在为命理咨询对话生成摘要。要求：\n" +
			"1. 保留所有命理术语原文（天干地支、十神、用神、日主、大运、流年、命宫、身宫、四化、格局、神煞、节气等），不得改写或意译。\n" +
			"2. 保留用户提供的出生信息（年月日时、性别）和**完整的**已排盘关键数据（四柱干支、十神、用神、大运、五行、命宫主星等）——命盘数据必须完整传递，不得截断、省略或改写。\n" +
			"3. 保留 specialist 的核心结论与判断依据。\n" +
			"4. 删除寒暄、过渡语、重复的盘面数据。\n" +
			"5. 摘要忠于原意，不添加推断或未提及的内容。",
	})
}

// knowledgeSearchMaxRunes 限制 knowledge_search 工具结果进入 LLM 上下文的最大长度（按 rune 计）。
// 古籍原文可达数千字，超长会挤占上下文并推高成本；超出部分直接丢弃，agent 只看到前段。
const knowledgeSearchMaxRunes = 2000

// buildToolReductionMiddleware 构造工具结果截断中间件。
// 仅对 knowledge_search 启用截断（per-tool TruncHandler），其他工具结果不受影响。
// 顶层 SkipTruncation + SkipClear 绕过 Backend 依赖；截断 handler 返回 NeedOffload=false，
// 不落盘、不注入 read_file 工具，超出部分直接丢弃。
func buildToolReductionMiddleware() (adk.ChatModelAgentMiddleware, error) {
	return reduction.New(context.Background(), &reduction.Config{
		SkipTruncation: true,
		SkipClear:      true,
		ToolConfig: map[string]*reduction.ToolReductionConfig{
			"knowledge_search": {
				SkipTruncation: false,
				TruncHandler:   knowledgeSearchTruncHandler,
			},
		},
	})
}

// knowledgeSearchTruncHandler 截断 knowledge_search 工具结果。
// 按 rune 计总长，超出 knowledgeSearchMaxRunes 时按出现顺序截断各文本 part，
// 末尾追加截断提示。非文本 part（图片/文件等）原样保留。不落盘。
func knowledgeSearchTruncHandler(_ context.Context, detail *reduction.ToolDetail) (*reduction.TruncResult, error) {
	if detail == nil || detail.ToolResult == nil {
		return &reduction.TruncResult{NeedTrunc: false}, nil
	}
	totalRunes := 0
	for _, part := range detail.ToolResult.Parts {
		if part.Type == schema.ToolPartTypeText {
			totalRunes += utf8.RuneCountInString(part.Text)
		}
	}
	if totalRunes <= knowledgeSearchMaxRunes {
		return &reduction.TruncResult{NeedTrunc: false}, nil
	}
	newParts := make([]schema.ToolOutputPart, len(detail.ToolResult.Parts))
	copy(newParts, detail.ToolResult.Parts)
	budget := knowledgeSearchMaxRunes
	for i := range newParts {
		if newParts[i].Type != schema.ToolPartTypeText {
			continue
		}
		r := []rune(newParts[i].Text)
		if len(r) > budget {
			newParts[i].Text = string(r[:budget]) + "\n…（古籍原文过长已截断，仅保留前段）"
			budget = 0
		} else {
			budget -= len(r)
		}
	}
	return &reduction.TruncResult{
		NeedTrunc:   true,
		ToolResult:  &schema.ToolResult{Parts: newParts},
		NeedOffload: false,
	}, nil
}

// buildBaziDataBlock 从会话状态中的 BaziResult 构建简明命盘数据摘要，注入 specialist instruction。
// buildYongshenBlock 从 yongshen 结果中提取关键字段，格局判定显式标注为确定性结论。
func buildYongshenBlock(sb *strings.Builder, ys map[string]interface{}) {
	sb.WriteString("**用神分析**：")
	if geju, ok := ys["geju"].(string); ok && geju != "" {
		sb.WriteString(fmt.Sprintf("格局=%s", geju))
		if status, ok := ys["geju_status"].(string); ok && status != "" {
			sb.WriteString(fmt.Sprintf("（%s）", status))
		}
		if detail, ok := ys["geju_detail"].(string); ok && detail != "" {
			sb.WriteString(fmt.Sprintf("，%s", detail))
		}
		if basis, ok := ys["geju_basis"].(string); ok && basis != "" {
			sb.WriteString(fmt.Sprintf("。取格依据：%s", basis))
		}
		if qz, ok := ys["geju_qing_zhuo"].(string); ok && qz != "" {
			sb.WriteString(fmt.Sprintf("。清浊：%s", qz))
		}
		if comb, ok := ys["geju_combination"].(string); ok && comb != "" && comb != "无明显组合关系" {
			sb.WriteString(fmt.Sprintf("。组合关系：%s", comb))
		}
		sb.WriteString("。**格局名为系统确定性计算结果。组合关系中的[主/次/辅/忌]为建议优先级（基于位置距离、半合局、身强身弱），你应综合全部事实做最终主次判断。**\n")
	}
	// 冲合刑害显式渲染：算法证据字段，LLM 在 interpret.md 命格层次清单中引用
	if ch, ok := ys["chonghe"].([]map[string]string); ok && len(ch) > 0 {
		parts := make([]string, 0, len(ch))
		for _, item := range ch {
			parts = append(parts, fmt.Sprintf("[%s]%s", item["type"], item["description"]))
		}
		sb.WriteString(fmt.Sprintf("**冲合刑害**：%s\n", strings.Join(parts, "；")))
	}
	// 十神力量显式渲染：按 weighted 降序，便于 LLM 识别主导十神
	if ssp, ok := ys["shi_shen_power"].(map[string]map[string]float64); ok && len(ssp) > 0 {
		type kv struct {
			god             string
			gan, zhi, total int
			weighted        float64
		}
		pairs := make([]kv, 0, len(ssp))
		for god, item := range ssp {
			pairs = append(pairs, kv{
				god:      god,
				gan:      int(item["gan_count"]),
				zhi:      int(item["zhi_count"]),
				total:    int(item["total"]),
				weighted: item["weighted"],
			})
		}
		sort.SliceStable(pairs, func(i, j int) bool {
			return pairs[i].weighted > pairs[j].weighted
		})
		parts := make([]string, 0, len(pairs))
		for _, p := range pairs {
			parts = append(parts, fmt.Sprintf("%s(透%d藏%d总%d权%.1f)", p.god, p.gan, p.zhi, p.total, p.weighted))
		}
		sb.WriteString(fmt.Sprintf("**十神力量**：%s\n", strings.Join(parts, " ")))
	}
	// 其余用神字段
	ysCopy := map[string]interface{}{}
	for k, v := range ys {
		if k != "geju" && k != "geju_status" && k != "geju_detail" && k != "geju_basis" && k != "geju_qing_zhuo" && k != "geju_combination" && k != "chonghe" && k != "shi_shen_power" {
			ysCopy[k] = v
		}
	}
	if j, err := json.Marshal(ysCopy); err == nil {
		sb.WriteString(fmt.Sprintf("**用神数据**：%s\n", string(j)))
	}
}

// buildYongshenBlockAny 是 buildYongshenBlock 的 map[string]any 版本。
func buildYongshenBlockAny(sb *strings.Builder, ys map[string]any) {
	m := make(map[string]interface{}, len(ys))
	for k, v := range ys {
		m[k] = v
	}
	buildYongshenBlock(sb, m)
}

// buildLiuNianBlock 渲染流年应期数据段。算法证据字段，LLM 在 interpret.md:140 框架下引用。
func buildLiuNianBlock(sb *strings.Builder, ln map[string]interface{}) {
	sb.WriteString("**流年应期**：")
	if year, ok := ln["liunian_year"]; ok {
		if gz, ok := ln["liunian_ganzhi"].(string); ok && gz != "" {
			sb.WriteString(fmt.Sprintf("%v年=%s。", year, gz))
		}
		if ss, ok := ln["liunian_shi_shen"].(string); ok && ss != "" {
			sb.WriteString(fmt.Sprintf("流年天干十神=%s。", ss))
		}
	}
	if cd, ok := ln["current_dayun"].(map[string]interface{}); ok && len(cd) > 0 {
		sb.WriteString(fmt.Sprintf("当前大运=%v-%v岁 %v。", cd["startAge"], cd["endAge"], cd["ganZhi"]))
	}
	if ch, ok := ln["liunian_chonghe"].([]map[string]string); ok && len(ch) > 0 {
		parts := make([]string, 0, len(ch))
		for _, item := range ch {
			parts = append(parts, fmt.Sprintf("[%s]%s", item["type"], item["description"]))
		}
		sb.WriteString(fmt.Sprintf("流年冲合刑害=%s。", strings.Join(parts, "；")))
	}
	sb.WriteString("\n")
}

// buildLiuNianBlockAny 是 buildLiuNianBlock 的 map[string]any 版本。
func buildLiuNianBlockAny(sb *strings.Builder, ln map[string]any) {
	m := make(map[string]interface{}, len(ln))
	for k, v := range ln {
		m[k] = v
	}
	buildLiuNianBlock(sb, m)
}

func (b *AgentBuilder) buildBaziDataBlock(st *state.SessionState) string {
	br := st.BaziResult
	if br == nil {
		return ""
	}

	var sb strings.Builder

	// 四柱十神
	sb.WriteString("**本命四柱十神（日柱标注日主，以下为命主本命，非大运）**：")
	if pillars, ok := br["pillars"].([]interface{}); ok {
		for _, p := range pillars {
			if pm, ok := p.(map[string]interface{}); ok {
				sb.WriteString(fmt.Sprintf(" %s%s(%s)",
					pm["stem"], pm["branch"], pm["shiShen"]))
			}
		}
	} else if pillars, ok := br["pillars"].([]map[string]any); ok {
		for _, pm := range pillars {
			sb.WriteString(fmt.Sprintf(" %s%s(%s)",
				pm["stem"], pm["branch"], pm["shiShen"]))
		}
	}
	sb.WriteString("\n")

	// 日主单独突出一行，防止 LLM 把大运干支误读为日主
	if dayGan, ok := br["dayGan"].(string); ok && dayGan != "" {
		sb.WriteString(fmt.Sprintf("**日主：%s（此为命主本命日干，大运干支不是日主）**\n", dayGan))
	}
	sb.WriteString("\n")

	// 五行统计
	if wx, ok := br["wuxing"].(map[string]interface{}); ok {
		sb.WriteString("**五行**：")
		for k, v := range wx {
			sb.WriteString(fmt.Sprintf(" %s:%v", k, v))
		}
		sb.WriteString("\n")
	} else if wx, ok := br["wuxing"].(map[string]int); ok {
		sb.WriteString("**五行**：")
		for k, v := range wx {
			sb.WriteString(fmt.Sprintf(" %s:%d", k, v))
		}
		sb.WriteString("\n")
	}

	// 用神
	if ys, ok := br["yongshen"].(map[string]interface{}); ok {
		buildYongshenBlock(&sb, ys)
	} else if ys, ok := br["yongshen"].(map[string]any); ok {
		buildYongshenBlockAny(&sb, ys)
	}

	// 大运
	if da, ok := br["dayun_analyzed"].(map[string]interface{}); ok {
		sb.WriteString("**大运分析**：")
		if j, err := json.Marshal(da); err == nil {
			sb.WriteString(string(j))
		}
		sb.WriteString("\n")
	} else if da, ok := br["dayun_analyzed"].(map[string]any); ok {
		sb.WriteString("**大运分析**：")
		if j, err := json.Marshal(da); err == nil {
			sb.WriteString(string(j))
		}
		sb.WriteString("\n")
	} else if dayun, ok := br["dayun"].([]interface{}); ok && len(dayun) > 0 {
		sb.WriteString("**大运脉络（以下为大运周期干支，非本命四柱，勿与日主混淆）**：")
		for i, d := range dayun {
			if dm, ok := d.(map[string]interface{}); ok {
				if i > 0 {
					sb.WriteString(" | ")
				}
				sb.WriteString(fmt.Sprintf("%v-%v岁 %s(大运)",
					dm["startAge"], dm["endAge"], dm["ganZhi"]))
			}
		}
		sb.WriteString("\n")
	}

	// 流年应期
	if ln, ok := br["liunian"].(map[string]interface{}); ok {
		buildLiuNianBlock(&sb, ln)
	} else if ln, ok := br["liunian"].(map[string]any); ok {
		buildLiuNianBlockAny(&sb, ln)
	}

	// 神煞
	if ss, ok := br["shensha_summary"].(map[string]interface{}); ok {
		if all, ok := ss["all"].([]interface{}); ok && len(all) > 0 {
			sb.WriteString("**主要神煞**：")
			for i, name := range all {
				if i > 0 {
					sb.WriteString("、")
				}
				sb.WriteString(fmt.Sprintf("%v", name))
			}
			sb.WriteString("\n")
		}
	}

	// 古籍背景知识
	if ks, ok := br["knowledge_summary"].(string); ok && ks != "" {
		sb.WriteString("\n**古籍背景知识**（盘面预检索，非针对当前问题）：\n")
		sb.WriteString(ks)
		sb.WriteString("\n")
	}

	// 兜底：完整 JSON
	if sb.Len() == 0 {
		if bj, err := json.Marshal(br); err == nil {
			sb.WriteString("<!-- 完整命盘 JSON（供推理引用，严禁逐项输出）\n")
			sb.WriteString(string(bj))
			sb.WriteString("\n-->\n")
		}
	}

	sb.WriteString("\n**\u26a0\ufe0f 以上数据均已就绪。忽略「执行规则」中的 1-3 步（排盘/用神/大运），直接用以上数据解读。如需特定古籍引用，可调用 knowledge_search。**\n")
	return sb.String()
}

// buildQimenDataBlock 从会话状态中的 QimenResult 构建简明奇门盘数据摘要。
func (b *AgentBuilder) buildQimenDataBlock(st *state.SessionState) string {
	qr := st.QimenResult
	if qr == nil {
		return ""
	}

	var sb strings.Builder

	if juText, ok := qr["ju_text"].(string); ok && juText != "" {
		sb.WriteString(fmt.Sprintf("**局数**：%s\n", juText))
	}
	if dutyStar, ok := qr["value_star"].(string); ok && dutyStar != "" {
		sb.WriteString(fmt.Sprintf("**值符星**：%s\n", dutyStar))
	}
	if dutyDoor, ok := qr["value_door"].(string); ok && dutyDoor != "" {
		sb.WriteString(fmt.Sprintf("**值使门**：%s\n", dutyDoor))
	}

	// 九宫信息
	if palaces, ok := qr["palaces"].([]interface{}); ok {
		sb.WriteString("**九宫**：")
		for _, p := range palaces {
			if pm, ok := p.(map[string]interface{}); ok {
				sb.WriteString(fmt.Sprintf(" %v(%v%v)",
					pm["name"], pm["star"], pm["door"]))
			}
		}
		sb.WriteString("\n")
	}

	// 兜底 JSON
	if sb.Len() == 0 {
		if bj, err := json.Marshal(qr); err == nil {
			sb.WriteString("<!-- 完整奇门盘 JSON（供推理引用）\n")
			sb.WriteString(string(bj))
			sb.WriteString("\n-->\n")
		}
	}

	sb.WriteString("\n**⚠️ 奇门盘数据已就绪，直接引用解读，禁止调用 qimen_dunjia。**\n")
	return sb.String()
}

// buildZiWeiDataBlock 从会话状态中的 ZiWeiResult 构建简明紫微命盘数据摘要。
func (b *AgentBuilder) buildZiWeiDataBlock(st *state.SessionState) string {
	zr := st.ZiWeiResult
	if zr == nil {
		return ""
	}

	var sb strings.Builder

	// 命宫主星
	if palaces, ok := zr["palaces"].([]interface{}); ok {
		for _, p := range palaces {
			if pm, ok := p.(map[string]interface{}); ok {
				name, _ := pm["name"].(string)
				if name == "命宫" || name == "身宫" {
					var stars []string
					if ms, ok := pm["major_stars"].([]interface{}); ok {
						for _, s := range ms {
							if sm, ok := s.(map[string]interface{}); ok {
								if sn, ok := sm["name"].(string); ok {
									stars = append(stars, sn)
								}
							}
						}
					}
					sb.WriteString(fmt.Sprintf("**%s主星**：%s\n", name, strings.Join(stars, "、")))
				}
			}
		}
	}

	// 四化
	if fp, ok := zr["four_pillars"].(map[string]interface{}); ok {
		if yp, ok := fp["年柱"].(string); ok && yp != "" {
			sb.WriteString(fmt.Sprintf("**生年年柱**：%s\n", yp))
		}
	}

	// 五行局
	if wx, ok := zr["wuxing_ju"].(string); ok && wx != "" {
		sb.WriteString(fmt.Sprintf("**五行局**：%s\n", wx))
	}

	// 流年
	if liunian, ok := zr["liunian"].(map[string]interface{}); ok {
		sb.WriteString("**流年数据**：")
		if j, err := json.Marshal(liunian); err == nil {
			sb.WriteString(string(j))
		}
		sb.WriteString("\n")
	}

	// 兜底 JSON
	if sb.Len() == 0 {
		if bj, err := json.Marshal(zr); err == nil {
			sb.WriteString("<!-- 完整紫微命盘 JSON（供推理引用）\n")
			sb.WriteString(string(bj))
			sb.WriteString("\n-->\n")
		}
	}

	sb.WriteString("\n**⚠️ 紫微命盘数据已就绪，直接引用解读，禁止调用 ziwei_calc/ziwei_liunian。**\n")
	return sb.String()
}

// Allowed returns a copy of the selected configs.
func (b *AgentBuilder) Allowed(route policy.ApprovedRoute, all []specialists.Config) []specialists.Config {
	return allowedSpecialists(route, all)
}

// BuildSupervisor 根据本轮批准路由构建 Supervisor Agent，只挂载允许调用的 AgentTool。
func (b *AgentBuilder) BuildSupervisor(ctx context.Context, route policy.ApprovedRoute, st *state.SessionState, allowedSpecialists []specialists.Config) (adk.Agent, error) {
	var agentTools []einotool.BaseTool
	for _, cfg := range allowedSpecialists {
		child, err := b.BuildSpecialist(ctx, cfg, st)
		if err != nil {
			return nil, fmt.Errorf("build specialist %s: %w", cfg.Name, err)
		}
		agt := adk.NewAgentTool(ctx, child)
		agentTools = append(agentTools, agt)
	}

	instruction := b.buildSupervisorInstruction(route, allowedSpecialists)

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "命理咨询执行主管，负责调度领域专家 Agent 完成分析。",
		Instruction: instruction,
		Model:       b.model,
		ToolsConfig: adk.ToolsConfig{
			EmitInternalEvents: true,
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentTools,
			},
		},
		MaxIterations:    10,
		GenModelInput:    noFStringGenModelInput,
		ModelRetryConfig: defaultRetryConfig(),
	})
}

// buildSupervisorInstruction 构建每轮动态 supervisor instruction。
func (b *AgentBuilder) buildSupervisorInstruction(route policy.ApprovedRoute, allowed []specialists.Config) string {
	toolDesc := formatAllowedTools(allowed)
	return fmt.Sprintf(`你是命理咨询执行主管。

## 身份
你不是权威路由器——权威路由已经由系统决策层完成。本轮批准的主领域是 %s，你只能调用下方可见的领域专家。

## 可见的领域专家（本轮允许调用）
%s

## 调用规则
1. 如果只有一个专家可见 → 直接调用它
2. 如果多个专家可见 → 先调主领域专家，再根据用户是否明确问了辅领域决定是否调第二个
3. 如果用户问题涉及多个领域但只有一个专家可见 → 只调可见的，不要抱怨缺少工具

## 最终回复规则
- 领域专家返回结果后，你的最终回复只需 1 句过渡引导（如"以上分析供您参考，如需追问可以继续。"）
- 不要重复、总结、缩写专家的分析内容
- 不要生成元描述（如"内容涵盖命盘总览、强弱调候..."）

## 禁止
- 不要回答命理分析问题（这由领域专家负责），你只做执行调度
- 不要请求更多工具或抱怨缺少工具
- 如果运行时 preflight 已放行 qimen-primary 且无 profile，不要追问出生信息`,
		route.PrimaryDomain,
		toolDesc,
	)
}

// formatAllowedTools 格式化可见 AgentTool 列表为 instruction 文本。
func formatAllowedTools(cfgs []specialists.Config) string {
	if len(cfgs) == 0 {
		return "（无可见专家）"
	}
	var b strings.Builder
	for i, cfg := range cfgs {
		b.WriteString(fmt.Sprintf("%d. **%s** - %s", i+1, cfg.Name, cfg.Description))
		if i < len(cfgs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// allowedSpecialists 根据 ApprovedRoute 过滤可用的 specialist 配置。
//
// 规则：
//   - 始终包含主域 specialist。
//   - qimen 仅在 QimenMode=primary 或 supplement 时包含。
//   - 其他辅域仅在 route.SecondaryDomains 明确包含时加入。
//   - fortune_followup + QimenMode=none 不包含 qimen。
//   - 未注册的域降级为 bazi。
func allowedSpecialists(route policy.ApprovedRoute, configs []specialists.Config) []specialists.Config {
	byDomain := make(map[string]specialists.Config)
	for _, cfg := range configs {
		byDomain[cfg.Domain] = cfg
	}

	// Determine primary domain, fallback to bazi
	primaryDomain := route.PrimaryDomain
	if _, ok := byDomain[primaryDomain]; !ok {
		primaryDomain = "bazi"
	}

	allowed := make(map[string]bool)
	allowed[primaryDomain] = true

	// 铁律：ziwei 作为 primary 时，bazi 必须可见
	if primaryDomain == "ziwei" {
		allowed["bazi"] = true
	}

	// Qimen visible only when QimenMode is primary or supplement
	if route.PolicyHints.QimenMode == "primary" || route.PolicyHints.QimenMode == "supplement" {
		allowed["qimen"] = true
	}

	// Other secondary domains explicit in the route
	for _, d := range route.SecondaryDomains {
		allowed[d] = true
	}

	// 婚姻/感情 queries → 自动包含 ziwei（作为辅域分析夫妻宫）
	if isMarriageQuery(route) && !allowed["ziwei"] {
		allowed["ziwei"] = true
	}

	var result []specialists.Config
	for _, cfg := range configs {
		if allowed[cfg.Domain] {
			result = append(result, cfg)
		}
	}
	// Always include primary domain even if not explicitly in allowed map
	found := false
	for _, cfg := range result {
		if cfg.Domain == primaryDomain {
			found = true
			break
		}
	}
	if !found {
		if cfg, ok := byDomain[primaryDomain]; ok {
			result = append(result, cfg)
		}
	}
	return result
}

// isMarriageQuery 检测用户是否在询问婚姻/感情相关主题。
// 如果触发，系统会自动将 ziwei 加入可见领域（作为辅域分析夫妻宫）。
func isMarriageQuery(route policy.ApprovedRoute) bool {
	subjects := []string{"婚姻", "感情", "夫妻", "配偶", "结婚", "恋爱", "合婚", "合不合适"}
	target := route.Slots.TargetSubject
	questionText := route.Slots.QuestionText
	for _, s := range subjects {
		if strings.Contains(target, s) || strings.Contains(questionText, s) {
			return true
		}
	}
	return false
}

// defaultRetryConfig 返回共享的 ModelRetryConfig，所有 Agent 统一使用。
func defaultRetryConfig() *adk.ModelRetryConfig {
	return &adk.ModelRetryConfig{
		MaxRetries: 2,
		ShouldRetry: func(ctx context.Context, rc *adk.RetryContext) *adk.RetryDecision {
			if rc.Err != nil {
				return &adk.RetryDecision{Retry: true, Backoff: time.Second}
			}
			return &adk.RetryDecision{Retry: false}
		},
	}
}
