package runtime

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"github.com/wikiglobal/suanming-agent/internal/tools"
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
type yongshenInput struct {
	DayMaster string `json:"day_master" jsonschema_description:"日主天干 (甲/乙/丙/丁/戊/己/庚/辛/壬/癸)"`
	Month     int    `json:"month" jsonschema_description:"出生月份 (1-12)，用于判断月令旺衰"`
	Gender    string `json:"gender" jsonschema_description:"性别 (男/女)"`
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

// newBaziCalcAdapter 创建八字排盘工具的 Eino BaseTool 适配器。
func newBaziCalcAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("bazi_calc", "计算八字排盘，输入出生年月日时+性别", func(ctx context.Context, input baziCalcInput) (string, error) {
		params, err := structToMap(input)
		if err != nil {
			return "{}", nil
		}
		gt, ok := reg.Get("bazi_calc")
		if !ok {
			return "{}", nil
		}
		result, err := gt.Execute(ctx, params)
		return marshalResult(result, err), nil
	})
}

// newYongshenAdapter 创建用神分析工具的 Eino BaseTool 适配器。
func newYongshenAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("yongshen", "分析日主强弱并推荐用神喜忌", func(ctx context.Context, input yongshenInput) (string, error) {
		params, err := structToMap(input)
		if err != nil {
			return "{}", nil
		}
		gt, ok := reg.Get("yongshen")
		if !ok {
			return "{}", nil
		}
		result, err := gt.Execute(ctx, params)
		return marshalResult(result, err), nil
	})
}

// newDayunAdapter 创建大运分析工具的 Eino BaseTool 适配器。
func newDayunAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("dayun_analyzer", "分析每个大运的吉凶和十神类型", func(ctx context.Context, input dayunInput) (string, error) {
		params, err := structToMap(input)
		if err != nil {
			return "{}", nil
		}
		gt, ok := reg.Get("dayun_analyzer")
		if !ok {
			return "{}", nil
		}
		result, err := gt.Execute(ctx, params)
		return marshalResult(result, err), nil
	})
}

// newQimenAdapter 创建奇门遁甲排盘工具的 Eino BaseTool 适配器。
func newQimenAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("qimen_dunjia", "奇门遁甲排盘，返回时家奇门九宫信息", func(ctx context.Context, input qimenInput) (string, error) {
		params, err := structToMap(input)
		if err != nil {
			return "{}", nil
		}
		gt, ok := reg.Get("qimen_dunjia")
		if !ok {
			return "{}", nil
		}
		result, err := gt.Execute(ctx, params)
		return marshalResult(result, err), nil
	})
}

// newZiweiAdapter 创建紫微斗数排盘工具的 Eino BaseTool 适配器。
func newZiweiAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("ziwei_calc", "紫微斗数排盘，输入出生年月日时+性别，返回命盘十二宫星曜布局", func(ctx context.Context, input ziweiInput) (string, error) {
		params, err := structToMap(input)
		if err != nil {
			return "{}", nil
		}
		gt, ok := reg.Get("ziwei_calc")
		if !ok {
			return "{}", nil
		}
		result, err := gt.Execute(ctx, params)
		return marshalResult(result, err), nil
	})
}

// newKnowledgeSearchAdapter 创建知识库检索工具的 Eino BaseTool 适配器。
func newKnowledgeSearchAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("knowledge_search", "检索项目知识库中的命理资料", func(ctx context.Context, input knowledgeSearchInput) (string, error) {
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
		b, e := json.Marshal(result)
		if e != nil {
			return `{"passages":[]}`, nil
		}
		return string(b), nil
	})
}

// newKnowledgeCatalogAdapter 创建知识库目录工具的 Eino BaseTool 适配器。
func newKnowledgeCatalogAdapter(reg *tools.Registry) (tool.BaseTool, error) {
	return utils.InferTool("knowledge_catalog",
		"获取知识库目录（古籍名称、章节数、前5个章节标题），用于规划检索",
		func(ctx context.Context, _ struct{}) (string, error) {
			gt, ok := reg.Get("knowledge_catalog")
			if !ok {
				return `{"error":"catalog not registered"}`, nil
			}
			result, err := gt.Execute(ctx, nil)
			return marshalResult(result, err), nil
		})
}


// buildAdapters 创建所有已注册命理工具的 Eino BaseTool 适配器列表。
//
// 对未注册的工具直接跳过，不返回错误。
// 返回的列表可直接注入到 Eino Agent 的 Tool 集合中。
func buildAdapters(reg *tools.Registry) ([]tool.BaseTool, error) {
	var adapters []tool.BaseTool

	entries := []struct {
		name    string
		builder func() (tool.BaseTool, error)
	}{
		{"bazi_calc", func() (tool.BaseTool, error) { return newBaziCalcAdapter(reg) }},
		{"yongshen", func() (tool.BaseTool, error) { return newYongshenAdapter(reg) }},
		{"dayun_analyzer", func() (tool.BaseTool, error) { return newDayunAdapter(reg) }},
		{"qimen_dunjia", func() (tool.BaseTool, error) { return newQimenAdapter(reg) }},
		{"ziwei_calc", func() (tool.BaseTool, error) { return newZiweiAdapter(reg) }},
		{"knowledge_search", func() (tool.BaseTool, error) { return newKnowledgeSearchAdapter(reg) }},
		{"knowledge_catalog", func() (tool.BaseTool, error) { return newKnowledgeCatalogAdapter(reg) }},
	}

	for _, entry := range entries {
		if _, ok := reg.Get(entry.name); !ok {
			continue
		}
		t, err := entry.builder()
		if err != nil {
			return nil, err
		}
		adapters = append(adapters, t)
	}

	return adapters, nil
}

// BuildAdaptersFor 根据请求的工具名称列表创建 Eino BaseTool 适配器。
//
// 未在 registry 中注册的工具会被静默跳过。
// 返回的适配器可直接注入到 Eino Agent 工具集合。
func BuildAdaptersFor(reg *tools.Registry, names []string) ([]tool.BaseTool, error) {
	builders := map[string]func() (tool.BaseTool, error){
		"bazi_calc":       func() (tool.BaseTool, error) { return newBaziCalcAdapter(reg) },
		"yongshen":        func() (tool.BaseTool, error) { return newYongshenAdapter(reg) },
		"dayun_analyzer":  func() (tool.BaseTool, error) { return newDayunAdapter(reg) },
		"qimen_dunjia":    func() (tool.BaseTool, error) { return newQimenAdapter(reg) },
		"ziwei_calc":      func() (tool.BaseTool, error) { return newZiweiAdapter(reg) },
		"knowledge_search": func() (tool.BaseTool, error) { return newKnowledgeSearchAdapter(reg) },
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
