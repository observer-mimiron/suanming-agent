// This file belongs to the session state layer.
// It owns domain context state for this package.
// It stores session truth; routing and interpretation decisions stay outside state structs.
package state

// ManagerContext 保存 manager 层持有的全局对话上下文。
type ManagerContext struct {
	ActiveDomain   string `json:"active_domain,omitempty"`
	CurrentTopic   string `json:"current_topic,omitempty"`
	WaitingOn      string `json:"waiting_on,omitempty"`
	LastReplyOwner string `json:"last_reply_owner,omitempty"`
}

// DomainContext 保存 specialist 层持有的领域执行上下文。
type DomainContext struct {
	Version        int            `json:"version,omitempty"`
	CheckpointID   string         `json:"checkpoint_id,omitempty"`
	InterruptID    string         `json:"interrupt_id,omitempty"`
	WorkingSummary string         `json:"working_summary,omitempty"`
	RuntimeValues  map[string]any `json:"runtime_values,omitempty"`
}

// LayeredDomainContexts 聚合各领域的执行上下文。
type LayeredDomainContexts struct {
	Bazi  DomainContext `json:"bazi,omitempty"`
	Qimen DomainContext `json:"qimen,omitempty"`
	ZiWei DomainContext `json:"ziwei,omitempty"`
}

func cloneDomainContext(ctx DomainContext) DomainContext {
	cloned := DomainContext{
		Version:        ctx.Version,
		CheckpointID:   ctx.CheckpointID,
		InterruptID:    ctx.InterruptID,
		WorkingSummary: ctx.WorkingSummary,
	}
	if len(ctx.RuntimeValues) > 0 {
		cloned.RuntimeValues = make(map[string]any, len(ctx.RuntimeValues))
		for k, v := range ctx.RuntimeValues {
			cloned.RuntimeValues[k] = v
		}
	}
	return cloned
}

func cloneLayeredDomainContexts(ctxs LayeredDomainContexts) LayeredDomainContexts {
	return LayeredDomainContexts{
		Bazi:  cloneDomainContext(ctxs.Bazi),
		Qimen: cloneDomainContext(ctxs.Qimen),
		ZiWei: cloneDomainContext(ctxs.ZiWei),
	}
}
