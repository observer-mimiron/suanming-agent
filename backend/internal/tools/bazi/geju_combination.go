package bazi

import (
	"fmt"
	"sort"
	"strings"
)

// detectGejuCombination 只报告十神组合候选及其所在位置。
// 组合同现不等于成格、富贵、为忌或主次；这些裁断必须由 selected rule profile
// 结合月令、透干、根气、救应和反证完成。
func detectGejuCombination(dayGan, dayWx, _ string, allGan, allZhi []string,
	stemWx map[string]string, branchHidegan map[string][]string, generates map[string]string) string {
	allGods := collectTenGods(dayGan, dayWx, allGan, allZhi, stemWx, generates, branchHidegan)

	hasGod := func(god string) bool {
		for _, info := range allGods {
			if info.god == god {
				return true
			}
		}
		return false
	}

	sources := func(gods ...string) string {
		selected := make([]tenGodInfo, 0)
		for _, info := range allGods {
			for _, god := range gods {
				if info.god == god {
					selected = append(selected, info)
					break
				}
			}
		}
		sort.SliceStable(selected, func(i, j int) bool {
			if selected[i].sourceWeight() != selected[j].sourceWeight() {
				return selected[i].sourceWeight() > selected[j].sourceWeight()
			}
			return selected[i].pillarIdx < selected[j].pillarIdx
		})
		values := make([]string, 0, len(selected))
		for _, info := range selected {
			values = append(values, info.god+"在"+info.displaySource())
		}
		return strings.Join(values, "、")
	}

	type candidate struct {
		name     string
		required []string
	}
	candidates := []candidate{
		{name: "食神制杀", required: []string{"食神", "七杀"}},
		{name: "杀印相生", required: []string{"七杀", "正印"}},
		{name: "杀印相生", required: []string{"七杀", "偏印"}},
		{name: "伤官佩印", required: []string{"伤官", "正印"}},
		{name: "伤官佩印", required: []string{"伤官", "偏印"}},
		{name: "官印相生", required: []string{"正官", "正印"}},
		{name: "官印相生", required: []string{"正官", "偏印"}},
		{name: "伤官生财", required: []string{"伤官", "正财"}},
		{name: "伤官生财", required: []string{"伤官", "偏财"}},
		{name: "食神生财", required: []string{"食神", "正财"}},
		{name: "食神生财", required: []string{"食神", "偏财"}},
		{name: "财官相生", required: []string{"正财", "正官"}},
		{name: "财官相生", required: []string{"偏财", "正官"}},
		{name: "财滋杀", required: []string{"正财", "七杀"}},
		{name: "财滋杀", required: []string{"偏财", "七杀"}},
		{name: "枭神夺食", required: []string{"偏印", "食神"}},
		{name: "伤官见官", required: []string{"伤官", "正官"}},
	}

	seen := map[string]bool{}
	results := make([]string, 0, len(candidates))
	for _, item := range candidates {
		valid := true
		for _, god := range item.required {
			if !hasGod(god) {
				valid = false
				break
			}
		}
		if !valid || seen[item.name] {
			continue
		}
		seen[item.name] = true
		results = append(results, fmt.Sprintf("%s候选（%s）", item.name, sources(item.required...)))
	}
	if len(results) == 0 {
		return "无十神组合候选"
	}
	return strings.Join(results, "；")
}
