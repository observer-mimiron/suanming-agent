package tracing

// ExecutionNode 是执行链路树中的一个节点。
// 叶子节点无 Children；父节点通过 Children 包含子步骤。
type ExecutionNode struct {
	Label    string         `json:"label"`
	Kind     SpanKind       `json:"kind"`
	Status   string         `json:"status"`
	Ms       int64          `json:"ms"`
	Meta     map[string]any `json:"meta,omitempty"`
	Children []ExecutionNode `json:"children,omitempty"`
}

// ExecutionTree 是从 TurnTrace 构建的统一执行链路树。
type ExecutionTree struct {
	TraceID  string        `json:"trace_id"`
	TurnType string        `json:"turn_type"`
	TotalMs  int64         `json:"total_ms"`
	Status   string        `json:"status"`
	Root     ExecutionNode `json:"root"`
}

// phaseGroup 定义语义阶段的分组规则。
type phaseGroup struct {
	names []string // 匹配的 span 名称
	label string   // 阶段中文标签
	order int      // 排序优先级
}

var phaseGroups = []phaseGroup{
	{names: []string{"supervisor_model", "output"}, label: "路由决策", order: 0},
	{names: []string{"preflight", "policy_gate"}, label: "执行前校验", order: 1},
	{names: []string{"prefill"}, label: "预填充复用", order: 2},
	{names: []string{"bazi_calc", "qimen_dunjia", "ziwei_calc", "ziwei_liunian", "yongshen", "dayun_analyzer"}, label: "命盘计算", order: 3},
	{names: []string{"knowledge_catalog", "knowledge_search"}, label: "知识检索", order: 4},
	{names: []string{"adk_supervisor_agent", "bazi_specialist", "qimen_specialist", "ziwei_specialist"}, label: "专家分析", order: 5},
	// sse_emit 被 compactSSEEmits 合并为 sse_emit_batch，归入 SSE 输出阶段
	{names: []string{"sse_emit_batch"}, label: "SSE 输出", order: 6},
}

// BuildExecutionTree 从 TurnTrace 构建统一的执行链路树。
//
// 步骤按语义阶段分组为嵌套树节点。连续的 sse_emit 通过 compactSSEEmits 合并后
// 归入 "SSE 输出" 阶段。未匹配任何阶段的步骤作为根节点叶子追加。
func (t *TurnTrace) BuildExecutionTree() ExecutionTree {
	// 1. 构建扁平 steps，compact SSE emits
	steps := make([]DebugTraceStep, 0, len(t.Spans))
	for _, s := range t.Spans {
		steps = append(steps, DebugTraceStep{
			Name:   s.Name,
			Label:  stepLabel(s.Name),
			Kind:   s.Kind,
			Status: normalizeStatus(s.Status),
			Ms:     s.DurationMs,
			Meta:   cloneAttrs(s.Attributes),
		})
	}
	steps = compactSSEEmits(steps)

	// 2. 将 steps 分组到语义阶段
	type phaseAcc struct {
		node  *ExecutionNode
		order int
	}
	phases := []phaseAcc{}
	phaseIdx := map[string]int{} // label → index in phases
	unmatched := []DebugTraceStep{}

	for _, s := range steps {
		matched := false
		for _, pg := range phaseGroups {
			if containsStr(pg.names, s.Name) {
				if idx, ok := phaseIdx[pg.label]; ok {
					// 追加到已有阶段
					n := phases[idx].node
					n.Children = append(n.Children, executionLeaf(s))
					n.Ms += s.Ms
					n.Status = worseStatus(n.Status, s.Status)
					mergeMeta(n.Meta, s.Meta, s.Kind)
				} else {
					// 新建阶段
					node := &ExecutionNode{
						Label:    pg.label,
						Kind:     KindChain,
						Status:   s.Status,
						Ms:       s.Ms,
						Meta:     map[string]any{},
						Children: []ExecutionNode{executionLeaf(s)},
					}
					mergeMeta(node.Meta, s.Meta, s.Kind)
					phaseIdx[pg.label] = len(phases)
					phases = append(phases, phaseAcc{node: node, order: pg.order})
				}
				matched = true
				break
			}
		}
		if !matched {
			unmatched = append(unmatched, s)
		}
	}

	// 3. 构建根节点
	root := ExecutionNode{
		Label:    "chat.turn",
		Kind:     KindAgent,
		Status:   normalizeStatus(t.Status),
		Ms:       traceTotalMs(t),
		Children: make([]ExecutionNode, 0, len(phases)+len(unmatched)),
	}

	// 4. 按 order 排序 phases
	sorted := make([]phaseAcc, len(phases))
	copy(sorted, phases)
	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && sorted[j].order < sorted[j-1].order {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			j--
		}
	}
	for _, p := range sorted {
		cleanNilMeta(p.node)
		root.Children = append(root.Children, *p.node)
	}

	// 5. 追加未匹配的叶子节点（如未来新增的 span type，作为安全网保留）
	for _, s := range unmatched {
		root.Children = append(root.Children, executionLeaf(s))
	}

	return ExecutionTree{
		TraceID:  t.TraceID,
		TurnType: t.TurnType,
		TotalMs:  traceTotalMs(t),
		Status:   normalizeStatus(t.Status),
		Root:     root,
	}
}

func executionLeaf(s DebugTraceStep) ExecutionNode {
	n := ExecutionNode{
		Label:  s.Label,
		Kind:   s.Kind,
		Status: s.Status,
		Ms:     s.Ms,
		Meta:   s.Meta,
	}
	if len(n.Meta) == 0 {
		n.Meta = nil
	}
	return n
}

func mergeMeta(dst map[string]any, src map[string]any, kind SpanKind) {
	if dst == nil {
		return
	}
	switch kind {
	case KindLLM:
		if v, ok := src["model"]; ok {
			dst["model"] = v
		}
		if v, ok := src["output_tokens"]; ok {
			dst["output_tokens"] = v
		}
	case KindRetriever:
		if v, ok := src["hits"]; ok {
			dst["hits"] = v
		}
		if v, ok := src["query"]; ok {
			dst["query"] = v
		}
		if v, ok := src["degrade_reason"]; ok {
			dst["degrade_reason"] = v
		}
	case KindTool:
		if v, ok := src["args"]; ok {
			dst["args"] = v
		}
		if v, ok := src["response"]; ok {
			dst["response"] = v
		}
	}
	if v, ok := src["batch_count"]; ok {
		dst["batch_count"] = v
	}
	if v, ok := src["breakdown"]; ok {
		dst["breakdown"] = v
	}
	if v, ok := src["degrade_reason"]; ok && dst["degrade_reason"] == nil {
		dst["degrade_reason"] = v
	}
	// 转发 thinking 文本到父阶段节点
	if v, ok := src["thinking"]; ok {
		dst["thinking"] = v
	}
}

func cleanNilMeta(n *ExecutionNode) {
	if n == nil {
		return
	}
	if len(n.Meta) == 0 {
		n.Meta = nil
	}
	if len(n.Children) == 0 {
		n.Children = nil
	}
	for i := range n.Children {
		cleanNilMeta(&n.Children[i])
	}
}

func containsStr(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
