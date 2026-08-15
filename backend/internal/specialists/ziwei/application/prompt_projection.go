// 本文件属于紫微 application 层。
// 本文件负责把已准备好的紫微命盘 payload 投影为 specialist instruction 数据块；
// 不负责读取 Session、调用模型/工具、写回状态、发送 trace/SSE 或生成最终用户文本。
package application

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildDataBlock 将紫微命盘 payload 按既有字段顺序投影为 specialist 输入。
// nil payload 返回空字符串；非 nil payload 保留旧的稀疏 JSON 兜底和禁止重复排盘提示。
func BuildDataBlock(zr map[string]any) string {
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

	// 生年年柱
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
