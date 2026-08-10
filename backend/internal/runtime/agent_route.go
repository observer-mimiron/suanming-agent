// Package runtime This file belongs to the manager-owned runtime layer.
// It owns agent route normalization for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
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
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/observer-mimiron/suanming-agent/internal/llm"
	"github.com/observer-mimiron/suanming-agent/internal/specialists"
	"github.com/observer-mimiron/suanming-agent/internal/state"
	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

// AgentBuilder 负责从 specialist 配置构建 ADK ChatModelAgent 和 AgentTool。
// 持有共享的 ToolCallingChatModel 和工具 Registry。
type AgentBuilder struct {
	model            einomodel.ToolCallingChatModel
	modelCreator     einomodel.ToolCallingChatModel // 主模型的 JSON Mode 变体
	fastModel        einomodel.ToolCallingChatModel
	fastModelCreator einomodel.ToolCallingChatModel // flash 模型的 JSON Mode 变体
	reg              *tools.Registry
	flashChat        llm.Chat
	summarizerModel  einomodel.ToolCallingChatModel
}

type AgentBuilderConfig struct {
	ModelCreator     einomodel.ToolCallingChatModel
	FastModel        einomodel.ToolCallingChatModel
	FastModelCreator einomodel.ToolCallingChatModel
}

// NewAgentBuilder 创建 AgentBuilder。
// summarizerModel 用于 summarization 中间件压缩长对话历史，通常应配置为成本较低的 flash 模型。
func NewAgentBuilder(model einomodel.ToolCallingChatModel, reg *tools.Registry, flashChat llm.Chat, summarizerModel einomodel.ToolCallingChatModel, cfg AgentBuilderConfig) *AgentBuilder {
	return &AgentBuilder{
		model:            model,
		modelCreator:     cfg.ModelCreator,
		fastModel:        cfg.FastModel,
		fastModelCreator: cfg.FastModelCreator,
		reg:              reg,
		flashChat:        flashChat,
		summarizerModel:  summarizerModel,
	}
}

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
	if cfg.InjectSessionContext && st != nil && (len(st.Profile) > 0 || st.HasBaziResult() || st.QimenResult != nil || st.HasZiWeiResult()) {
		sessionCtx = "## 会话已有上下文\n\n以下资料已在当前会话中提供，**直接使用，无需再次索要或调用工具获取**：\n"
		if len(st.Profile) > 0 {
			sessionCtx += "\n### 出生资料\n" + buildProfileSection(st) + "\n"
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
	if cfg.StructuredSchema != "" {
		contract, err := structuredOutputPromptContract(cfg.StructuredSchema)
		if err != nil {
			return nil, err
		}
		instruction += "\n\n" + contract
	}
	if sessionCtx != "" {
		if strings.Contains(instruction, sessionCtxPlaceholder) {
			instruction = strings.Replace(instruction, sessionCtxPlaceholder, sessionCtx, 1)
		} else {
			instruction = sessionCtx + "\n\n---\n\n" + instruction
		}
	}
	instruction += "\n\n## 当前运行时上下文\n当前系统日期：" + time.Now().Format("2006-01-02") + "。\n当前系统时区：Asia/Shanghai。"
	agentModel := b.model
	if cfg.UseFastModel && b.fastModel != nil {
		agentModel = b.fastModel
	}
	if cfg.UseJSONMode {
		if cfg.UseFastModel && b.fastModelCreator != nil {
			agentModel = b.fastModelCreator
		} else if b.modelCreator != nil {
			agentModel = b.modelCreator
		}
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Instruction:   instruction,
		Model:         agentModel,
		MaxIterations: 15,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: adapters,
			},
		},
		Handlers:         b.buildSpecialistHandlers(),
		GenModelInput:    noFStringGenModelInput,
		ModelRetryConfig: llm.DefaultModelRetryConfig(),
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
// 当前阈值放宽到 8000 rune：正常长度的古籍摘录尽量完整保留，仅在明显超长时再截断，
// 避免证据被过早裁短到只剩碎片。
const knowledgeSearchMaxRunes = 8000

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
// buildYongshenBlock 从 yongshen 结果中提取受力与结构候选；它们不是最终命理结论。
func buildYongshenBlock(sb *strings.Builder, ys map[string]interface{}) {
	sb.WriteString("**八字结构事实与候选**：")
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
		if reason, ok := ys["geju_qing_zhuo_reason"].(map[string]interface{}); ok && len(reason) > 0 {
			if summary, ok := reason["summary"].(string); ok && summary != "" {
				sb.WriteString(fmt.Sprintf("。清浊依据：%s", summary))
			}
			switch ev := reason["evidence"].(type) {
			case []string:
				if len(ev) > 0 {
					sb.WriteString(fmt.Sprintf("。证据：%s", strings.Join(ev, "；")))
				}
			case []interface{}:
				parts := make([]string, 0, len(ev))
				for _, item := range ev {
					if s, ok := item.(string); ok && s != "" {
						parts = append(parts, s)
					}
				}
				if len(parts) > 0 {
					sb.WriteString(fmt.Sprintf("。证据：%s", strings.Join(parts, "；")))
				}
			}
		}
		if comb, ok := ys["geju_combination"].(string); ok && comb != "" && comb != "无明显组合关系" {
			sb.WriteString(fmt.Sprintf("。组合关系：%s", comb))
		}
		sb.WriteString("。**格局名、强弱、用忌和组合关系均须读取其 method/profile：排盘与藏干位置可复算，组合标签只表示结构候选或受阻事实，不能直接等同成格。**\n")
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
		if k != "geju" && k != "geju_status" && k != "geju_detail" && k != "geju_basis" && k != "geju_qing_zhuo" && k != "geju_qing_zhuo_reason" && k != "geju_combination" && k != "chonghe" && k != "shi_shen_power" {
			ysCopy[k] = v
		}
	}
	if j, err := json.Marshal(ysCopy); err == nil {
		sb.WriteString(fmt.Sprintf("**用神数据**：%s\n", string(j)))
	}
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

	// 五行统计（明面八位：四干四支主气，不含藏干）
	if wx, ok := br["wuxing"].(map[string]interface{}); ok {
		sb.WriteString("**五行（明面八位，不含藏干）**：")
		for k, v := range wx {
			sb.WriteString(fmt.Sprintf(" %s:%v", k, v))
		}
		sb.WriteString("\n")
	} else if wx, ok := br["wuxing"].(map[string]int); ok {
		sb.WriteString("**五行（明面八位，不含藏干）**：")
		for k, v := range wx {
			sb.WriteString(fmt.Sprintf(" %s:%d", k, v))
		}
		sb.WriteString("\n")
	}

	// 用神
	if ys, ok := br["yongshen"].(map[string]interface{}); ok {
		buildYongshenBlock(&sb, ys)
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
	}

	// 神煞
	appendShenshaPromptBlock(&sb, br["shensha_summary"])

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

type promptShenshaSummary struct {
	All      []string                       `json:"all"`
	ByPillar map[string][]promptShenshaItem `json:"by_pillar"`
}

type promptShenshaItem struct {
	Name        string `json:"name"`
	Tone        string `json:"tone"`
	Basis       string `json:"basis"`
	Description string `json:"description"`
}

func appendShenshaPromptBlock(sb *strings.Builder, raw any) {
	if raw == nil {
		return
	}

	var summary promptShenshaSummary
	payload, err := json.Marshal(raw)
	if err != nil {
		return
	}
	if err := json.Unmarshal(payload, &summary); err != nil {
		return
	}

	wroteAny := false
	if len(summary.All) > 0 {
		sb.WriteString("**主要神煞**：")
		for i, name := range summary.All {
			if i > 0 {
				sb.WriteString("、")
			}
			sb.WriteString(name)
		}
		sb.WriteString("\n")
		wroteAny = true
	}

	pillarOrder := []string{"年柱", "月柱", "日柱", "时柱"}
	pillarLines := make([]string, 0, len(pillarOrder))
	for _, pillar := range pillarOrder {
		items := summary.ByPillar[pillar]
		if len(items) == 0 {
			continue
		}

		parts := make([]string, 0, len(items))
		for _, item := range items {
			parts = append(parts, fmt.Sprintf("%s[%s/%s]", item.Name, shenshaToneLabel(item.Tone), item.Basis))
		}
		pillarLines = append(pillarLines, fmt.Sprintf("%s：%s", pillar, strings.Join(parts, "、")))
	}
	if len(pillarLines) > 0 {
		sb.WriteString("**按柱神煞**：")
		sb.WriteString(strings.Join(pillarLines, "；"))
		sb.WriteString("\n")
		wroteAny = true
	}

	if wroteAny {
		sb.WriteString("**神煞使用口径**：先看月令、旺衰、格局、用神，再看十神、生克制化、刑冲合害，最后才把神煞作为辅助佐证；只重点解释高频、低争议、与用户问题直接相关的神煞；只有当神煞与原局结构、大运、流年同向印证时才强化判断，若与主线冲突则主动降权处理；禁止仅凭单一神煞直接断必然吉凶或下灾祸定论。\n")
	}
}

func shenshaToneLabel(tone string) string {
	switch tone {
	case "good":
		return "吉"
	case "bad":
		return "凶"
	default:
		return "平"
	}
}

// buildQimenDataBlock 从会话状态中的 QimenResult 构建简明奇门盘数据摘要。
func (b *AgentBuilder) buildQimenDataBlock(st *state.SessionState) string {
	qr := st.QimenResult
	if qr == nil {
		return ""
	}

	var sb strings.Builder

	if value := stringValue(qr["case_id"]); value != "" {
		sb.WriteString(fmt.Sprintf("**Case**：%s\n", value))
	}
	if value := stringValue(qr["purpose"]); value != "" {
		sb.WriteString(fmt.Sprintf("**问事目的**：%s\n", value))
	}
	if owner, ok := qr["owner_ref"].(map[string]any); ok {
		sb.WriteString(fmt.Sprintf("**资产归属**：%s/%s\n", stringValue(owner["kind"]), stringValue(owner["id"])))
	}
	if value := stringValue(qr["question_time"]); value != "" {
		sb.WriteString(fmt.Sprintf("**提问时间**：%s\n", value))
	}
	if value := stringValue(qr["time_source"]); value != "" {
		sb.WriteString(fmt.Sprintf("**起局时间来源**：%s\n", value))
	}
	if value := stringValue(qr["symbol_system"]); value != "" {
		sb.WriteString(fmt.Sprintf("**符号体系**：%s\n", value))
	}

	if juText, ok := qr["ju_text"].(string); ok && juText != "" {
		sb.WriteString(fmt.Sprintf("**局数**：%s\n", juText))
	}
	if dutyStar, ok := qr["value_star"].(string); ok && dutyStar != "" {
		sb.WriteString(fmt.Sprintf("**值符星**：%s\n", dutyStar))
	}
	if dutyDoor, ok := qr["value_door"].(string); ok && dutyDoor != "" {
		sb.WriteString(fmt.Sprintf("**值使门**：%s\n", dutyDoor))
	}
	if schema := stringValue(qr["pan_schema"]); schema != "" {
		sb.WriteString(fmt.Sprintf("**盘式口径**：%s\n", schema))
	}
	if palace := stringValue(qr["duty_star_palace"]); palace != "" {
		sb.WriteString(fmt.Sprintf("**值符宫**：%s\n", palace))
	}
	if palace := stringValue(qr["duty_door_palace"]); palace != "" {
		sb.WriteString(fmt.Sprintf("**值使宫**：%s\n", palace))
	}

	// 九宫信息
	if cells, ok := qr["cells"].([]interface{}); ok {
		sb.WriteString("**九宫**：")
		for _, p := range cells {
			if pm, ok := p.(map[string]interface{}); ok {
				sb.WriteString(fmt.Sprintf(" %v(%v星/%v门/%v神/天%v地%v)",
					pm["palace"], pm["star"], pm["door"], pm["god"], pm["guest_gan"], pm["host_gan"]))
			}
		}
		sb.WriteString("\n")
	} else if cells, ok := qr["cells"].([]map[string]any); ok {
		sb.WriteString("**九宫**：")
		for _, pm := range cells {
			sb.WriteString(fmt.Sprintf(" %v(%v星/%v门/%v神/天%v地%v)",
				pm["palace"], pm["star"], pm["door"], pm["god"], pm["guest_gan"], pm["host_gan"]))
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
