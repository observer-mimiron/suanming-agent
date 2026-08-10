// Package specialists 定义领域专家的静态配置和执行契约。
// 当前 runtime 仍可继续消费 Config 构建 specialist agent，
// 同时 registry 只保留 runner 映射，避免保留一份不会被读取的配置副本。
package specialists

// Config 描述一个领域专家的静态元数据。
// 这些字段仍由 runtime.AgentBuilder 消费，用于构建现有 specialist agent。
type Config struct {
	Domain      string
	Name        string
	Description string
	Instruction string
	ToolNames   []string
	// InjectSessionContext 控制是否把 SESSION_CONTEXT 形式的整包会话上下文自动注入 instruction。
	// 外层 specialist 默认保持开启；authority-first 内层 graph 节点会显式关闭，改为消费阶段化视图。
	InjectSessionContext bool
	// UseJSONMode 启用模型级 JSON Mode（response_format: json_object）。仅对需要结构化 JSON 输出的 agent 节点启用；
	// 当前纯八字最终成文已改为 Go 端程序渲染，不再依赖自由文本 writer。
	UseJSONMode bool
	// UseFastModel 允许该节点优先使用 flash 级模型。
	// 适合分析模式判定、证据规划这类轻判断节点，不适合承载完整命理综合推理的主节点。
	UseFastModel bool
	// StructuredSchema 标识此节点唯一的结构化输出合同。空值表示自由文本或工具调用。
	// runtime 负责将同一份 Schema 注入 prompt 并在模型返回后校验，specialist 不解释合同内容。
	StructuredSchema string
}

// Registry 保存当前运行时可用的 specialist runner。
type Registry struct {
	runners map[string]Runner
}

// NewRegistry 创建空 Registry。
func NewRegistry() *Registry {
	return &Registry{runners: make(map[string]Runner)}
}

// Register 记录一个领域对应的 runner。
func (r *Registry) Register(cfg Config, runner Runner) {
	if r == nil || cfg.Domain == "" || runner == nil {
		return
	}
	r.runners[cfg.Domain] = runner
}

// RunnerFor 返回指定领域的 runner。
func (r *Registry) RunnerFor(domain string) (Runner, bool) {
	if r == nil {
		return nil, false
	}
	runner, ok := r.runners[domain]
	return runner, ok
}
