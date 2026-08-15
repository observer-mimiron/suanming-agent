// Package domain 包含不需要 runtime 服务的八字确定性计算。
//
// 本文件负责八字合同与投影共用的文本列表归一；
// 不读取会话，不调用模型、检索、追踪或输出传输。
package domain

import "strings"

// NonEmptyStrings 去除空白条目并保持原有顺序。
func NonEmptyStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// UniqueStrings 去除完全相同的文本，保留首次出现顺序。
func UniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range NonEmptyStrings(items) {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
