// Package specialists 定义了领域专家的通用接口和类型。
// 专家模式中每个 Specialist 通过 Config + Registry 注册其 AgentTool 元数据，
// 由 runtime.AgentBuilder 构建 ADK ChatModelAgent。
package specialists

// Config 描述一个领域专家 AgentTool 的静态元数据，由对应的 specialist 包注册到 Registry。
// 不依赖 runtime 包，避免循环引用。
type Config struct {
	Domain      string
	Name        string
	Description string
	Instruction string
	ToolNames   []string
}

// Registry 存储领域专家配置，按注册顺序保存。用于构建 AgentTool 列表。
type Registry struct {
	configs []Config
}

// NewRegistry 创建空 Registry。
func NewRegistry() *Registry { return &Registry{} }

// Register 追加一个领域专家配置。
func (r *Registry) Register(cfg Config) { r.configs = append(r.configs, cfg) }

// All 返回已注册的 Config 切片的副本。
func (r *Registry) All() []Config {
	out := make([]Config, len(r.configs))
	copy(out, r.configs)
	return out
}
