package orchestrator

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/policy"
	"github.com/wikiglobal/suanming-agent/internal/state"
)


// classifyAndExtract uses LLM to determine user intent and extract birth info.
// Returns: action, profile patch, question text, needsQimen, rawBazi.
// Actions: "new_profile" | "update_profile" | "followup" | "incomplete" | "bazi_input"
func (o *Orchestrator) classifyAndExtract(ctx context.Context, msg string, st *state.SessionState) (string, map[string]any, string, bool, []string) {
	profileJSON, _ := json.Marshal(st.Profile)

	prompt := `你是一个信息提取助手。分析用户输入，输出纯JSON。

## 可能的动作

- **bazi_input**: 用户直接提供了四柱八字（如"乙巳 丁亥 甲申 甲子"），不需要排盘，直接分析这个八字
- **new_profile**: 用户提供了新的出生信息（年份+月份+日期），要排一个新命盘
- **update_profile**: 用户纠正或修改已有信息（如"不对，我是女的"、"改成1991年"）
- **followup**: 用户针对已有命盘提问、追问（如"今年运势怎么样"、"此命1996年发生何事"）
- **incomplete**: 出生信息仍然不完整，需要继续追问

## 判断规则

1. **出生时间优先**：如果输入同时包含出生时间（年份+月份+日期）和四柱八字 → 优先 new_profile。仅有性别、地点等附加信息但没有年份+月份+日期的，不算"出生时间"，仍按纯八字处理
2. **纯八字**：如果输入包含四柱八字格式，但没有完整的出生时间（即使附带了性别、地点等其他信息）→ bazi_input。对于 bazi_input，仍然提取消息中的 gender 字段（如有）
3. 如果输入包含 年份+月份+日期（完整出生时间），且不是针对已有命盘的提问 → new_profile
4. 如果输入是纠正/修改信息 → update_profile，只提取变化的字段。**重要：当已有命盘或已有部分资料时，单独的「男」「女」「北京」等字段补全都属于 update_profile，绝不要判为 new_profile**
5. 如果已有完整资料，且输入是提问/追问 → followup，question=用户原始输入
6. 如果资料不完整且输入不是出生信息 → incomplete

## 提取规则 (仅 new_profile / update_profile 时)

- year: 数字 (1900-2100)。如果输入没有年份，不要提取
- month: 数字 (1-12)
- day: 数字 (1-31)
- hour: 24小时制数字 (0-23)
  - 上午→0-11，下午/晚上→12-23
  - 时辰→子时23 丑时1 寅时3 卯时5 辰时7 巳时9 午时11 未时13 申时15 酉时17 戌时19 亥时21
- minute: 数字 (0-59)。如果用户明确说了分钟（如"23点40分"、"8:12"），提取分钟；否则不填
- gender: "男" 或 "女"
- calendar_type: "solar"(公历) 或 "lunar"(农历)
- birthplace: 出生城市/地区（如"北京"、"广东"、"美国纽约"）。如用户提及则提取，否则留空
- longitude: 出生地经度，数字。如果用户明确说了地点，尝试估算。中国主要城市参考：北京116.4, 上海121.5, 广州113.3, 成都104.1, 乌鲁木齐87.6, 哈尔滨126.6。海外城市需用户明确说明。不填时默认120.0（不做太阳时校正）

## 当前日期
今天是 ` + time.Now().Format("2006-01-02") + `（请以此日期判断"今天""今年""本月"等时间词）

## 当前已有资料
` + string(profileJSON) + `

## needs_qimen 判断

如果用户问题涉及**当前时间段**（本月、今年、最近、今天、近期、当下、这一阵子等），且询问的是时机/运势/适合与否，设置 needs_qimen: true。问具体某一年（如1996年如何）或问性格/事业方向等长期问题的，设置 false。

## needs_knowledge 判断

**大部分追问都应该设为 true。** 知识检索开销很低，而命理典籍引用能显著提升回答质量。以下情况设为 false：
- 纯闲聊（"你好""谢谢"）
- 用户明确要求不引用资料
- 追问的内容已在前一轮充分引用过，且新一轮只是确认细节

其他所有涉及命理分析、运势判断、性格解读、婚姻/事业/健康/财运/流年的追问，一律设为 true。

## 只返回JSON，不要任何其他文字

{"action":"<action>","year":1990,"month":5,"day":20,"hour":8,"gender":"男","calendar_type":"solar","birthplace":"北京","longitude":116.4,"needs_qimen":false,"needs_knowledge":true,"question":"用户的原始问题"}`

	messages := []llm.Message{
		{Role: "user", Content: msg},
	}

	resp, _, err := o.flash.Generate(ctx, prompt, messages)
	if err != nil {
		st.NeedsKnowledge = true
		patch, q := extractProfileAndQuestion(msg)
		if st.HasBaziResult() || st.IsProfileComplete() {
			return "followup", nil, msg, false, nil
		}
		return "incomplete", patch, q, false, nil
	}

	var result struct {
		Action       string  `json:"action"`
		Year         float64 `json:"year"`
		Month        float64 `json:"month"`
		Day          float64 `json:"day"`
		Hour         float64 `json:"hour"`
		Minute       float64 `json:"minute"`
		Gender       string  `json:"gender"`
		CalendarType string  `json:"calendar_type"`
		Birthplace   string  `json:"birthplace"`
		Longitude    float64 `json:"longitude"`
		NeedsQimen     bool    `json:"needs_qimen"`
		NeedsKnowledge bool    `json:"needs_knowledge"`
		Question       string  `json:"question"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &result); err != nil {
		st.NeedsKnowledge = true
		patch, q := extractProfileAndQuestion(msg)
		if st.HasBaziResult() || st.IsProfileComplete() {
			return "followup", nil, msg, false, nil
		}
		return "incomplete", patch, q, false, nil
	}
	st.NeedsKnowledge = result.NeedsKnowledge

	patch := map[string]any{}
	if result.Year >= 1900 && result.Year <= 2100 {
		patch["year"] = result.Year
	}
	if result.Month >= 1 && result.Month <= 12 {
		patch["month"] = result.Month
	}
	if result.Day >= 1 && result.Day <= 31 {
		patch["day"] = result.Day
	}
	if result.Hour >= 0 && result.Hour <= 23 {
		patch["hour"] = result.Hour
	}
	if result.Minute >= 0 && result.Minute <= 59 {
		patch["minute"] = result.Minute
	}
	if result.Gender == "男" || result.Gender == "女" {
		patch["gender"] = result.Gender
	}
	if result.CalendarType != "" {
		patch["calendar_type"] = result.CalendarType
	} else {
		patch["calendar_type"] = "solar"
	}
	if result.Birthplace != "" {
		patch["birthplace"] = result.Birthplace
	}
	if result.Longitude >= -180 && result.Longitude <= 180 {
		patch["longitude"] = result.Longitude
	}

	action := result.Action
	if action == "" {
		action = "followup"
	}
	question := strings.TrimSpace(result.Question)
	if question == "" {
		question = msg
	}
	// For bazi_input, extract the 4 gan-zhi pairs from the message using regex
	var rawBazi []string
	if action == "bazi_input" {
		rawBazi = extractBaziPillars(msg)
	}
	return action, patch, question, result.NeedsQimen, rawBazi
}

// baziPairRe matches a single gan-zhi pair like "甲申" or "乙巳".
var baziPairRe = regexp.MustCompile(`([甲乙丙丁戊己庚辛壬癸][子丑寅卯辰巳午未申酉戌亥])`)

// extractBaziPillars extracts the first 4 gan-zhi pairs from a message.
func extractBaziPillars(msg string) []string {
	matches := baziPairRe.FindAllString(msg, -1)
	if len(matches) < 4 {
		return nil
	}
	return matches[:4]
}

// bridgeDecision converts a policy-approved route into the legacy action-based routing tuple.
// It must receive the ApprovedRoute (post-policy-gate), not the raw SupervisorDecision,
// so that low-confidence clarification, domain downgrades, and other policy overrides
// are enforced in the live turn flow.
func bridgeDecision(route policy.ApprovedRoute, msg string) (action string, patch map[string]any, question string, needsQimen bool, rawBazi []string) {
	patch = route.Slots.Profile
	if patch == nil {
		patch = map[string]any{}
	}
	question = route.Slots.QuestionText
	if question == "" {
		question = msg
	}
	needsQimen = route.PolicyHints.NeedsQimen

	// If the policy gate forced clarification, short-circuit to incomplete.
	if route.NeedsClarification {
		return "incomplete", patch, question, needsQimen, rawBazi
	}

	switch route.TaskIntent {
	case "collect_profile":
		action = "new_profile"
	case "amend_profile":
		action = "update_profile"
	case "direct_bazi":
		action = "bazi_input"
		rawBazi = extractBaziPillars(msg)
	case "interpret_chart":
		action = "followup"
	case "fortune_followup":
		action = "followup"
		needsQimen = true
	case "timing_followup", "cross_domain_consult":
		action = "followup"
		needsQimen = true
	default:
		action = "followup"
	}

	return action, patch, question, needsQimen, rawBazi
}
