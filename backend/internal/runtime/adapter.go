// This file belongs to the manager-owned runtime layer.
// It owns runtime adapter behavior for this package.
// It owns execution contracts and Manager flow; specialists do not own final answers.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/llm"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/observer-mimiron/suanming-agent/internal/tools"
)

// baziCalcInput 八字排盘工具的输入参数。
type baziCalcInput struct {
	Year   int     `json:"year" jsonschema_description:"出生年份 (1900-2100)"`
	Month  int     `json:"month" jsonschema_description:"出生月份 (1-12)"`
	Day    int     `json:"day" jsonschema_description:"出生日期 (1-31)"`
	Hour   float64 `json:"hour" jsonschema_description:"出生时辰 (0-23, 24小时制)"`
	Gender string  `json:"gender" jsonschema_description:"性别 (男/女)"`
}

// yongshenInput 用神分析工具的输入参数。
// 需要完整的年月日时+性别才能用 lunar-go 排盘后推算用神。
type yongshenInput struct {
	Year   int    `json:"year" jsonschema_description:"出生年份 (1900-2100)"`
	Month  int    `json:"month" jsonschema_description:"出生月份 (1-12)"`
	Day    int    `json:"day" jsonschema_description:"出生日期 (1-31)"`
	Hour   int    `json:"hour" jsonschema_description:"出生时辰 (0-23)"`
	Gender string `json:"gender" jsonschema_description:"性别 (男/女)"`
}

// dayunInput 大运分析工具的输入参数。
type dayunInput struct {
	BaziJSON string `json:"bazi_json" jsonschema_description:"八字排盘结果的 JSON 字符串，包含 pillars、dayGan、dayZhi、dayun 等字段"`
	Gender   string `json:"gender" jsonschema_description:"性别 (男/女)"`
}

// qimenInput 奇门遁甲排盘工具的输入参数。
type qimenInput struct {
	Year   int    `json:"year" jsonschema_description:"年份"`
	Month  int    `json:"month" jsonschema_description:"月份"`
	Day    int    `json:"day" jsonschema_description:"日期"`
	Hour   int    `json:"hour" jsonschema_description:"时辰 (0-23)"`
	Minute int    `json:"minute,omitempty" jsonschema_description:"分钟 (0-59)"`
	TermID int    `json:"term_id,omitempty" jsonschema_description:"节气 ID"`
	Ju     string `json:"ju,omitempty" jsonschema_description:"局数"`
}

// ziweiInput 紫微斗数排盘工具的输入参数。
type ziweiInput struct {
	Year   int    `json:"year" jsonschema_description:"出生年份"`
	Month  int    `json:"month" jsonschema_description:"出生月份 (1-12)"`
	Day    int    `json:"day" jsonschema_description:"出生日期 (1-31)"`
	Hour   int    `json:"hour" jsonschema_description:"出生时辰 (0-23)"`
	Gender string `json:"gender" jsonschema_description:"性别 (男/女)"`
	Leap   bool   `json:"leap,omitempty" jsonschema_description:"是否闰月"`
}
type ziweiLiuNianInput struct {
	Year       int    `json:"year" jsonschema_description:"出生年份"`
	Month      int    `json:"month" jsonschema_description:"出生月份 (1-12)"`
	Day        int    `json:"day" jsonschema_description:"出生日期 (1-31)"`
	Hour       int    `json:"hour" jsonschema_description:"出生时辰 (0-23)"`
	Gender     string `json:"gender" jsonschema_description:"性别 (男/女)"`
	TargetYear int    `json:"target_year" jsonschema_description:"流年目标年份"`
	Age        int    `json:"age" jsonschema_description:"虚岁年龄"`
}

// knowledgeSearchInput 知识库检索工具的输入参数。
type knowledgeSearchInput struct {
	Query string `json:"query" jsonschema_description:"搜索查询文本"`
	TopK  int    `json:"top_k" jsonschema_description:"返回结果数量 (默认 5)"`
}

// structToMap 将任意结构体转为 map[string]any，用于工具参数传递。
func structToMap(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// marshalResult 将工具执行结果序列化为 JSON 字符串，错误或空结果返回空对象。
func marshalResult(result any, err error) string {
	if err != nil || result == nil {
		return "{}"
	}
	b, e := json.Marshal(result)
	if e != nil {
		return "{}"
	}
	return string(b)
}

func executeRegistryTool(ctx context.Context, reg *tools.Registry, name string, params map[string]any) string {
	gt, ok := reg.Get(name)
	if !ok {
		return "{}"
	}
	result, err := gt.Execute(ctx, params)
	return marshalResult(result, err)
}

func inferRegistryTool[I any](reg *tools.Registry, name, desc string) (tool.BaseTool, error) {
	return utils.InferTool(name, desc, func(ctx context.Context, input I) (string, error) {
		params, err := structToMap(input)
		if err != nil {
			return "{}", nil
		}
		return executeRegistryTool(ctx, reg, name, params), nil
	})
}

// newBaziCalcAdapter 创建八字排盘工具的 Eino BaseTool 适配器。
func newBaziCalcAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return inferRegistryTool[baziCalcInput](reg, "bazi_calc", "计算八字排盘，输入出生年月日时+性别")
}

// newYongshenAdapter 创建用神分析工具的 Eino BaseTool 适配器。
func newYongshenAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return inferRegistryTool[yongshenInput](reg, "yongshen", "分析日主强弱并推荐用神喜忌")
}

// newDayunAdapter 创建大运分析工具的 Eino BaseTool 适配器。
//
// 将 LLM 传入的 bazi_json 字符串解析后：
//  1. 提取 dayun 数组和 bazi_result 对象
//  2. 若 yongshen 缺失，从 bazi_json 的 birthday 字段反推出生时间，调用 yongshen 工具兜底
//  3. 将补全后的 bazi_result 传给 dayun_analyzer
func newDayunAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("dayun_analyzer", "分析每个大运的吉凶和十神类型", func(ctx context.Context, input dayunInput) (string, error) {
		params, err := structToMap(input)
		if err != nil {
			return "{}", nil
		}
		baziJSON, _ := params["bazi_json"].(string)
		gender, _ := params["gender"].(string)
		if baziJSON == "" {
			return "{}", nil
		}
		var baziResult map[string]any
		if err := json.Unmarshal([]byte(baziJSON), &baziResult); err != nil {
			return "{}", nil
		}
		if gender != "" && baziResult["gender"] == nil {
			baziResult["gender"] = gender
		}

		// 若 yongshen 缺失，从 birthday 字段反推出生时间并调用 yongshen 工具兜底
		if baziResult["yongshen"] == nil {
			if yt, ok := reg.Get("yongshen"); ok {
				if yongshenParams := buildYongshenParamsFromBaziResult(baziResult); yongshenParams != nil {
					if yr, yerr := yt.Execute(ctx, yongshenParams); yerr == nil && yr != nil {
						if ym, ok := yr.(map[string]any); ok {
							baziResult["yongshen"] = ym
						}
					}
				}
			}
		}

		toolParams := map[string]any{
			"dayun":       baziResult["dayun"],
			"bazi_result": baziResult,
		}
		return executeRegistryTool(ctx, reg, "dayun_analyzer", toolParams), nil
	})
}

// buildYongshenParamsFromBaziResult 从 bazi_result 的 birthday 字段反推出 yongshen 工具所需的参数。
//
// birthday 格式为 "YYYY-MM-DD HH:MM"，如 "2020-10-10 10:00"。
// 返回 nil 表示无法解析。
func buildYongshenParamsFromBaziResult(baziResult map[string]any) map[string]any {
	birthday, _ := baziResult["birthday"].(string)
	if birthday == "" {
		return nil
	}
	// 解析 "2020-10-10 10:00"
	var year, month, day, hour int
	n, _ := fmt.Sscanf(birthday, "%d-%d-%d %d", &year, &month, &day, &hour)
	if n < 4 {
		return nil
	}
	if year < 1900 || year > 2100 || month < 1 || month > 12 || day < 1 || day > 31 || hour < 0 || hour > 23 {
		return nil
	}
	gender, _ := baziResult["gender"].(string)
	if gender == "" {
		gender = "男"
	}
	return map[string]any{
		"year":   float64(year),
		"month":  float64(month),
		"day":    float64(day),
		"hour":   float64(hour),
		"gender": gender,
	}
}

// newQimenAdapter 创建奇门遁甲排盘工具的 Eino BaseTool 适配器。
func newQimenAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return inferRegistryTool[qimenInput](reg, "qimen_dunjia", "奇门遁甲排盘，返回时家奇门九宫信息")
}

// newZiweiAdapter 创建紫微斗数排盘工具的 Eino BaseTool 适配器。
func newZiweiAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return inferRegistryTool[ziweiInput](reg, "ziwei_calc", "紫微斗数排盘，输入出生年月日时+性别，返回命盘十二宫星曜布局")
}

// newZiweiLiuNianAdapter 创建紫微斗数流年分析工具的 Eino BaseTool 适配器。
func newZiweiLiuNianAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return inferRegistryTool[ziweiLiuNianInput](reg, "ziwei_liunian", "紫微斗数流年分析，输入出生年月日时+性别+目标年份+虚岁年龄")
}

// newKnowledgeSearchAdapter 创建知识库检索工具的 Eino BaseTool 适配器。
func newKnowledgeSearchAdapter(reg *tools.Registry, flashChat llm.Chat) (tool.BaseTool, error) {
	var callCount int // 闭包计数器，随 adapter 生命周期（per-turn）归零

	return utils.InferTool("knowledge_search", "在命理古籍知识库中检索原文。每轮限3次调用，请在前3次内覆盖所有关键词。query 使用核心术语，优先用典籍名+章节名限定范围。", func(ctx context.Context, input knowledgeSearchInput) (string, error) {
		callCount++
		if callCount > 3 {
			return `{"passages":[],"budget_exceeded":true,"message":"本轮知识检索已达上限（3次），请基于已有资料回答。"}`, nil
		}
		params, err := structToMap(input)
		if err != nil {
			return `{"passages":[]}`, nil
		}
		gt, ok := reg.Get("knowledge_search")
		if !ok {
			return `{"passages":[]}`, nil
		}
		result, err := gt.Execute(ctx, params)
		if err != nil || result == nil {
			return `{"passages":[]}`, nil
		}
		// 压缩古籍段落：优先用 flash 模型提炼要点，fallback 到 400 字截断
		if rm, ok := result.(map[string]any); ok {
			if passages, ok := rm["passages"].([]interface{}); ok && len(passages) > 3 {
				rm["passages"] = passages[:3]
			}
			if flashChat != nil {
				if compressed := compressPassagesWithFlash(ctx, flashChat, rm); compressed != nil {
					for k, v := range compressed {
						rm[k] = v
					}
				}
			} else {
				truncatePassages(rm, 400)
			}
		}
		b, e := json.Marshal(result)
		if e != nil {
			return `{"passages":[]}`, nil
		}
		return string(b), nil
	})
}

// catalogInput is a dummy input struct for the knowledge_catalog tool.
// Eino InferTool requires input structs to have at least one exported field.
type catalogInput struct {
	_ struct{}
}

// newKnowledgeCatalogAdapter 创建知识库目录工具的 Eino BaseTool 适配器。
func newKnowledgeCatalogAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	var callCount int // 闭包计数器，目录只需获取一次
	return utils.InferTool("knowledge_catalog",
		"获取知识库目录（古籍名称、章节数、前5个章节标题），用于规划检索。限1次调用，超出自动阻断——请基于已有目录信息调 knowledge_search 检索。",
		func(ctx context.Context, _ catalogInput) (string, error) {
			callCount++
			if callCount > 1 {
				return "目录已在上一条 knowledge_catalog 工具结果中提供。你现在必须调用 knowledge_search 检索古籍原文，不要再调 knowledge_catalog。", nil
			}
			if _, ok := reg.Get("knowledge_catalog"); !ok {
				return `{"error":"catalog not registered"}`, nil
			}
			return executeRegistryTool(ctx, reg, "knowledge_catalog", nil), nil
		})
}

// BuildAdaptersFor 根据请求的工具名称列表创建 Eino BaseTool 适配器。
//
// 未在 registry 中注册的工具会被静默跳过。
// 返回的适配器可直接注入到 Eino Agent 工具集合。
func BuildAdaptersFor(reg *tools.Registry, names []string, flashChat llm.Chat) ([]tool.BaseTool, error) {
	builders := map[string]func() (tool.BaseTool, error){
		"bazi_calc":         func() (tool.BaseTool, error) { return newBaziCalcAdapter(reg) },
		"yongshen":          func() (tool.BaseTool, error) { return newYongshenAdapter(reg) },
		"dayun_analyzer":    func() (tool.BaseTool, error) { return newDayunAdapter(reg) },
		"qimen_dunjia":      func() (tool.BaseTool, error) { return newQimenAdapter(reg) },
		"ziwei_calc":        func() (tool.BaseTool, error) { return newZiweiAdapter(reg) },
		"ziwei_liunian":     func() (tool.BaseTool, error) { return newZiweiLiuNianAdapter(reg) },
		"knowledge_search":  func() (tool.BaseTool, error) { return newKnowledgeSearchAdapter(reg, flashChat) },
		"knowledge_catalog": func() (tool.BaseTool, error) { return newKnowledgeCatalogAdapter(reg) },
	}
	adapters := make([]tool.BaseTool, 0, len(names))
	for _, name := range names {
		if _, ok := reg.Get(name); !ok {
			continue
		}
		build, ok := builders[name]
		if !ok {
			continue
		}
		t, err := build()
		if err != nil {
			return nil, err
		}
		adapters = append(adapters, t)
	}
	return adapters, nil
}

// compressPassagesWithFlash 用 flash 模型将古籍原文段落压缩为要点摘要，保留出处。
func compressPassagesWithFlash(ctx context.Context, flashChat llm.Chat, rm map[string]any) map[string]any {
	passages, ok := rm["passages"].([]interface{})
	if !ok || len(passages) == 0 {
		return nil
	}
	var sb strings.Builder
	for _, p := range passages {
		if pm, ok := p.(map[string]interface{}); ok {
			if src, ok := pm["source"].(string); ok {
				sb.WriteString("【" + src + "】")
			}
			if c, ok := pm["content"].(string); ok {
				sb.WriteString(c)
			}
		}
		sb.WriteString("\n---\n")
	}
	body := sb.String()
	if len(body) > 6000 {
		body = body[:6000]
	}
	prompt := "将以下古籍原文提炼为关键命理要点，保留典籍出处（用【书名】标注），每条不超过80字。只输出要点，不要解释。"
	summary, _, err := flashChat.Generate(ctx, prompt, []llm.Message{{Role: "user", Content: body}})
	if err != nil || summary == "" {
		return nil // fallback to truncation
	}
	return map[string]any{"summary": summary, "passages": passages}
}

// truncatePassages 将 passages 中的 content 截断到 maxLen 字符。
func truncatePassages(rm map[string]any, maxLen int) {
	passages, ok := rm["passages"].([]interface{})
	if !ok {
		return
	}
	for _, p := range passages {
		if pm, ok := p.(map[string]interface{}); ok {
			if c, ok := pm["content"].(string); ok && len(c) > maxLen {
				pm["content"] = c[:maxLen] + "..."
			}
		}
	}
}
