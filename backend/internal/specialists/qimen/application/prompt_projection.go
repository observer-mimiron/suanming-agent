// 本文件属于奇门 application 层。
// 本文件负责把已准备好的奇门盘面 payload 投影为 specialist instruction 数据块；
// 不负责读取 Session、调用模型/工具、写回状态、发送 trace/SSE 或生成最终用户文本。
package application

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildDataBlock 将当前 Case 的奇门盘面 payload 按既有顺序投影为 specialist 输入。
// nil payload 返回空字符串；非 nil payload 的字段缺失时保留旧的省略和 JSON 兜底行为。
func BuildDataBlock(qr map[string]any) string {
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

	// 九宫信息必须兼容 JSON 解码后的 []interface{} 和测试/内部组装的 []map[string]any。
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

	// 兜底 JSON 保留旧 payload 的完整可引用内容，避免稀疏结果变成无上下文提示。
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
