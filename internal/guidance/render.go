// Package guidance 负责把轻量对话引导指令渲染为面向用户的中文提示语。
package guidance

import (
	"strings"

	"github.com/wikiglobal/suanming-agent/internal/schemas"
	"github.com/wikiglobal/suanming-agent/internal/state"
)

// BoundaryPrompt 标识 execution-boundary 里的确定性提示模板。
type BoundaryPrompt string

const (
	// BoundaryCollectGenderFromBirthTime 用于“已拿到出生时间但缺性别”的确定性追问。
	BoundaryCollectGenderFromBirthTime BoundaryPrompt = "collect_gender_from_birth_time"
	// BoundaryClarificationFallback 用于 supervisor 明确要求澄清时的统一兜底提示。
	BoundaryClarificationFallback BoundaryPrompt = "clarification_fallback"
	// BoundaryAskFullProfile 用于需要完整出生资料的通用提示。
	BoundaryAskFullProfile BoundaryPrompt = "ask_full_profile"
	// BoundaryAskBaziProfile 用于八字主链缺资料时的提示。
	BoundaryAskBaziProfile BoundaryPrompt = "ask_bazi_profile"
	// BoundaryAskZiweiProfile 用于紫微主链缺资料时的提示。
	BoundaryAskZiweiProfile BoundaryPrompt = "ask_ziwei_profile"
)

// Context 只承载 renderer 做文案选择时真正需要的最小上下文。
type Context struct {
	ClarificationQuestion string
}

// Request 描述一次 guidance 文案渲染请求。
type Request struct {
	Directive *schemas.ConversationDirective
	Boundary  BoundaryPrompt
	Context   Context
}

// Render 根据 directive 或 boundary prompt 产出最终用户可见文案。
func Render(req Request) string {
	if req.Directive != nil {
		return renderDirective(*req.Directive, req.Context)
	}
	return renderBoundary(req.Boundary, req.Context)
}

// RenderGuidance 从 GuidanceState 渲染用户可见文案（code-owned 路径入口）。
func RenderGuidance(gs *state.GuidanceState, ctx Context) string {
	if gs == nil {
		return renderGuidedFallback(ctx)
	}
	switch gs.DirectiveKind {
	case "offer_consult":
		return "如果您愿意，我可以按命理咨询的方式继续帮您看。您可以直接告诉我这次最想重点了解什么，比如事业、感情、财运、健康或流年。"
	case "choose_topic":
		return renderChooseTopic("top_topics")
	case "collect_slot":
		return renderCollectSlot(gs.PendingSlot)
	case "guided_fallback":
		return renderGuidedFallback(ctx)
	default:
		return renderGuidedFallback(ctx)
	}
}

// RenderBoundary 是 renderBoundary 的公开入口，供 runtime 直接调用。
func RenderBoundary(boundary BoundaryPrompt, ctx Context) string {
	return renderBoundary(boundary, ctx)
}

func renderDirective(d schemas.ConversationDirective, ctx Context) string {
	switch d.Kind {
	case "offer_consult":
		return "如果您愿意，我可以按命理咨询的方式继续帮您看。您可以直接告诉我这次最想重点了解什么，比如事业、感情、财运、健康或流年。"
	case "choose_topic":
		return renderChooseTopic(d.OptionSet)
	case "collect_slot":
		return renderCollectSlot(d.SlotName)
	case "guided_fallback":
		return renderGuidedFallback(ctx)
	default:
		return renderGuidedFallback(ctx)
	}
}

func renderBoundary(boundary BoundaryPrompt, ctx Context) string {
	switch boundary {
	case BoundaryCollectGenderFromBirthTime:
		return "已经收到您的出生时间，还需要确认一项关键信息：性别是男还是女？\n\n八字排盘的大运顺逆排法因性别而异，请告知后再为您完整分析。"
	case BoundaryClarificationFallback:
		return renderGuidedFallback(ctx)
	case BoundaryAskFullProfile:
		return "需要通过完整的出生信息进行综合分析。请提供出生年月日时和性别。"
	case BoundaryAskBaziProfile:
		return "请提供您的出生信息（年份、月份、日期、时辰、性别），我来为您排盘分析。"
	case BoundaryAskZiweiProfile:
		return "需要出生信息才能排紫微斗数命盘。请提供出生年月日时和性别。"
	default:
		return renderGuidedFallback(Context{})
	}
}

func renderChooseTopic(optionSet string) string {
	switch optionSet {
	case "top_topics", "":
		return "我可以先从几个常见方向来帮您看：事业、感情、财运、健康、流年。您这次最想先聊哪一项？"
	default:
		return "我可以继续帮您细化方向。您这次最想先聊哪一项？"
	}
}

func renderCollectSlot(slotName string) string {
	switch slotName {
	case "gender":
		return "还差一个关键信息：性别。请告诉我是男还是女。"
	case "birth_time":
		return "还差一个关键信息：出生时辰。请告诉我大概几点出生；如果只记得范围，也可以直接告诉我。"
	default:
		return "还差一个关键信息。请再补充一下，我就可以继续为您分析。"
	}
}

func renderGuidedFallback(ctx Context) string {
	question := strings.TrimSpace(ctx.ClarificationQuestion)
	if question != "" {
		return question
	}
	return "请确认一下您的需求，我再为您详细分析。"
}
