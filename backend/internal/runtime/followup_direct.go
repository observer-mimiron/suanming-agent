package runtime

import (
	"strings"

	"github.com/observer-mimiron/suanming-agent/internal/policy"
	"github.com/observer-mimiron/suanming-agent/internal/state"
)

type baziGlossaryEntry struct {
	needles []string
	answer  string
}

var baziGlossaryEntries = []baziGlossaryEntry{
	{
		needles: []string{"财星破印", "财星坏印", "财坏印", "财破印", "破印", "坏印"},
		answer:  "财星破印，意思是财来克印。财星一旺，印星原本的护持、承接、缓冲之力容易被压住，所以古法会说“财破印”或“财坏印”。这本身是通用术语，不等于一看到这四个字就能直接重判整盘高低；放回具体命盘，还要看它是在讲主轴、限制，还是某一步岁运触发。",
	},
	{
		needles: []string{"食神制杀"},
		answer:  "食神制杀，意思是用食神去制七杀，把原本偏刚、偏压的杀气收束成可用之力。成立的关键不是名字好听，而是要看食神有没有根气、能不能真制得住七杀；若力量不够，就只能算方向可见，不一定能当成强主轴。",
	},
	{
		needles: []string{"杀印相生", "印化杀"},
		answer:  "杀印相生，意思是七杀的压力通过印星承接、转化，最后不直接伤身，反而变成可用之力。这个术语成立时，重点要看印星是否真能承接七杀，而不是只要盘里同时见杀和印，就直接判成杀印相生。",
	},
	{
		needles: []string{"伤官配印"},
		answer:  "伤官配印，意思是伤官的表达、泄秀、抗衡之力，能被印星收住并转成较稳的才华、学识或输出能力。重点不在“同时见伤官和印”，而在印能不能把伤官的散与冲收回来。",
	},
	{
		needles: []string{"弃印就财"},
		answer:  "弃印就财，意思是原局印路不足以成主轴，反而顺着财路去看结构与出路。它不是简单地“有财就好”，而是说明这盘在取舍上更偏向财路，而不是继续守印路。",
	},
	{
		needles: []string{"月劫格", "劫财格"},
		answer:  "月劫格，意思是月令之气以劫财为主，所以先按比劫框架看这张盘的结构。它回答的是“这盘按什么框架立局”，不等于已经把成局路线、层次高低、岁运兑现都一起定完。",
	},
	{
		needles: []string{"用神"},
		answer:  "用神，意思是这张盘里最关键的着力点：哪一种五行或十神最能解决核心矛盾、扶住主轴。它不是“喜欢什么就补什么”的口语版，而是要结合格局、调候、扶抑和病药一起定。",
	},
	{
		needles: []string{"忌神"},
		answer:  "忌神，意思是会继续放大这张盘核心矛盾、病点或结构冲突的那股力量。它不是单纯“不喜欢”，而是“再加上去会更坏事”的那一边，所以要看它到底在放大什么问题。",
	},
	{
		needles: []string{"调候"},
		answer:  "调候，意思是先看寒暖燥湿这一层环境条件是否对，再谈格局和扶抑。它回答的是“这盘最先要解决的气候问题是什么”，所以和单纯的旺衰喜忌不是一回事。",
	},
	{
		needles: []string{"扶抑"},
		answer:  "扶抑，意思是看日主偏强还是偏弱，再决定该扶还是该抑。它主要回答受力平衡问题，和调候、格局是不同维度，不能互相直接替代。",
	},
}

// maybeDirectBaziGlossaryFollowup 在 manager-owned preflight 层拦截“纯术语解释型追问”，
// 避免这类不依赖具体命盘重判的 follow-up 再次进入八字 graph。
func maybeDirectBaziGlossaryFollowup(st *state.SessionState, route policy.ApprovedRoute, message string) (string, bool) {
	if st == nil || !st.HasBaziResult() {
		return "", false
	}
	if route.PrimaryDomain != "bazi" || route.TaskIntent != "fortune_followup" {
		return "", false
	}
	if len(route.SecondaryDomains) > 0 || route.PolicyHints.QimenMode == "primary" || route.PolicyHints.QimenMode == "supplement" {
		return "", false
	}

	trimmed := strings.TrimSpace(message)
	if trimmed == "" || !isBaziGlossaryExplainQuestion(trimmed) || isChartDependentBaziQuestion(trimmed) {
		return "", false
	}

	for _, entry := range baziGlossaryEntries {
		if containsAnyText([]string{trimmed}, entry.needles) {
			return entry.answer, true
		}
	}
	return "", false
}

func isBaziGlossaryExplainQuestion(message string) bool {
	return containsAnyText([]string{message}, []string{
		"啥意思", "什么意思", "怎么理解", "解释一下", "解释下", "是啥", "是什么", "什么叫", "指什么", "怎么读",
	})
}

func isChartDependentBaziQuestion(message string) bool {
	return containsAnyText([]string{message}, []string{
		"我这盘", "我这命", "我这个命", "我这里", "在我这里", "这盘里", "在这盘里",
		"原局里", "在原局里", "我的八字", "我的命盘", "我的盘", "我这个八字",
		"为什么算", "为什么我", "为什么说我", "为什么在我这里", "那我这盘", "那我这里",
	})
}
