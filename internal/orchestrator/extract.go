package orchestrator

import (
	"encoding/json"
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/llm"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// classifyAndExtract uses LLM to determine user intent and extract birth info.
// Returns: action, profile patch, question text.
// Actions: "new_profile" | "update_profile" | "followup" | "incomplete"
func classifyAndExtract(msg string, st *state.SessionState) (string, map[string]any, string) {
	profileJSON, _ := json.Marshal(st.Profile)

	prompt := `你是一个信息提取助手。分析用户输入，输出纯JSON。

## 可能的动作

- **new_profile**: 用户提供了新的出生信息（年份+月份+日期），要排一个新命盘
- **update_profile**: 用户纠正或修改已有信息（如"不对，我是女的"、"改成1991年"）
- **followup**: 用户针对已有命盘提问、追问（如"今年运势怎么样"、"此命1996年发生何事"）
- **incomplete**: 出生信息仍然不完整，需要继续追问

## 判断规则

1. 如果输入包含 年份+月份+日期（完整出生时间），且不是针对已有命盘的提问 → new_profile
2. 如果输入是纠正/修改信息 → update_profile，只提取变化的字段
3. 如果已有完整资料，且输入是提问/追问 → followup，question=用户原始输入
4. 如果资料不完整 → incomplete

## 提取规则 (仅 new_profile / update_profile 时)

- year: 数字 (1900-2100)。如果输入没有年份，不要提取
- month: 数字 (1-12)
- day: 数字 (1-31)
- hour: 24小时制数字 (0-23)
  - 上午→0-11，下午/晚上→12-23
  - 时辰→子时23 丑时1 寅时3 卯时5 辰时7 巳时9 午时11 未时13 申时15 酉时17 戌时19 亥时21
- gender: "男" 或 "女"
- calendar_type: "solar"(公历) 或 "lunar"(农历)

## 当前已有资料
` + string(profileJSON) + `

## 只返回JSON，不要任何其他文字

{"action":"<action>","year":1990,"month":5,"day":20,"hour":8,"gender":"男","calendar_type":"solar","question":"用户的原始问题"}`

	messages := []llm.Message{
		{Role: "user", Content: msg},
	}

	client := llm.NewClient()
	resp, err := client.Chat(prompt, messages)
	if err != nil {
		// Fall back to regex
		patch, q := extractProfileAndQuestion(msg)
		if st.IsProfileComplete() {
			return "followup", nil, msg
		}
		return "incomplete", patch, q
	}

	var result struct {
		Action       string  `json:"action"`
		Year         float64 `json:"year"`
		Month        float64 `json:"month"`
		Day          float64 `json:"day"`
		Hour         float64 `json:"hour"`
		Gender       string  `json:"gender"`
		CalendarType string  `json:"calendar_type"`
		Question     string  `json:"question"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &result); err != nil {
		patch, q := extractProfileAndQuestion(msg)
		if st.IsProfileComplete() {
			return "followup", nil, msg
		}
		return "incomplete", patch, q
	}

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
	if result.Gender == "男" || result.Gender == "女" {
		patch["gender"] = result.Gender
	}
	if result.CalendarType != "" {
		patch["calendar_type"] = result.CalendarType
	} else {
		patch["calendar_type"] = "solar"
	}

	action := result.Action
	if action == "" {
		action = "followup"
	}
	question := strings.TrimSpace(result.Question)
	if question == "" {
		question = msg
	}
	return action, patch, question
}
