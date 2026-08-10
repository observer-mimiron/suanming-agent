// Package structured 提供跨运行层共享的 JSON Schema 注册、prompt 注入与严格解码。
//
// 它只强制结构合同，不了解路由、命理事实、修复策略、渲染或 SSE。
package structured

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

var registry = struct {
	sync.RWMutex
	schemas map[string][]byte
}{schemas: map[string][]byte{}}

const draft07SchemaURI = "http://json-schema.org/draft-07/schema#"

// Error 表示 JSON 外形、Schema 或严格 DTO 解码失败。
type Error struct{ Schema, Detail string }

func (e *Error) Error() string { return fmt.Sprintf("schema_error[%s]: %s", e.Schema, e.Detail) }

// RegisterBundle 注册仓库内的 JSON Schema bundle，并在启动时验证每条 Schema 自身符合 Draft-07。
// bundle 的 key 是合同名，value 是供 prompt 与客户端校验共同消费的原始 Schema。
func RegisterBundle(bundle []byte) error {
	var schemas map[string]json.RawMessage
	if err := json.Unmarshal(bundle, &schemas); err != nil {
		return fmt.Errorf("decode structured schema bundle: %w", err)
	}
	if len(schemas) == 0 {
		return fmt.Errorf("structured schema bundle is empty")
	}
	for name, raw := range schemas {
		if err := validateSchemaDocument(name, raw); err != nil {
			return err
		}
		if err := registerRawSchema(name, raw); err != nil {
			return err
		}
	}
	return nil
}

// RegisterJSON 注册一份独立 JSON Schema，适用于不共享 bundle 文件的边界组件。
func RegisterJSON(name string, raw []byte) error {
	if err := validateSchemaDocument(name, raw); err != nil {
		return err
	}
	return registerRawSchema(name, raw)
}

// validateSchemaDocument 检查维护者提交的 Schema，而不是等到模型返回时才发现合同文件损坏。
func validateSchemaDocument(name string, raw []byte) error {
	loader := gojsonschema.NewSchemaLoader()
	loader.AutoDetect = false
	loader.Draft = gojsonschema.Draft7
	loader.Validate = true
	if _, err := loader.Compile(gojsonschema.NewStringLoader(string(raw))); err != nil {
		return fmt.Errorf("invalid Draft-07 schema %q: %w", name, err)
	}
	return nil
}

// registerRawSchema 保存 Schema 原文；重复注册必须字节级一致，避免出现隐式合同覆盖。
func registerRawSchema(name string, raw []byte) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("structured schema name is empty")
	}
	copyRaw := append([]byte(nil), raw...)
	registry.Lock()
	defer registry.Unlock()
	if prior, ok := registry.schemas[name]; ok {
		if !bytes.Equal(prior, copyRaw) {
			return fmt.Errorf("structured schema %q already registered with another contract", name)
		}
		return nil
	}
	registry.schemas[name] = copyRaw
	return nil
}

// PromptContract returns the exact registered Schema used later for validation.
func PromptContract(name string) (string, error) {
	raw, err := schema(name)
	if err != nil {
		return "", err
	}
	return "## 结构化输出合同\n你只能输出一个 JSON object，不得使用 Markdown fence 或额外文本。以下 Schema 是唯一字段合同；不得自行新增字段、事实值、来源、recovery、audit 或 renderer 字段。\nSchema 名称：" + name + "\nJSON Schema:\n" + string(raw), nil
}

// Hash exposes the exact Schema fingerprint for regression tests.
func Hash(name string) (string, error) {
	raw, err := schema(name)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// Decode validates raw JSON then requires an exact single DTO value.
func Decode(name, raw string, target any) error {
	if strings.TrimSpace(raw) == "" {
		return &Error{Schema: name, Detail: "empty output"}
	}
	if strings.Contains(raw, "```") {
		return &Error{Schema: name, Detail: "markdown fence is not JSON"}
	}
	rawSchema, err := schema(name)
	if err != nil {
		return &Error{Schema: name, Detail: err.Error()}
	}
	result, err := gojsonschema.Validate(gojsonschema.NewStringLoader(string(rawSchema)), gojsonschema.NewStringLoader(raw))
	if err != nil {
		return &Error{Schema: name, Detail: err.Error()}
	}
	if !result.Valid() {
		details := make([]string, 0, len(result.Errors()))
		for _, issue := range result.Errors() {
			details = append(details, issue.String())
		}
		return &Error{Schema: name, Detail: strings.Join(details, "; ")}
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &Error{Schema: name, Detail: err.Error()}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return &Error{Schema: name, Detail: "trailing JSON value"}
		}
		return &Error{Schema: name, Detail: "trailing data: " + err.Error()}
	}
	return nil
}

func schema(name string) ([]byte, error) {
	registry.RLock()
	defer registry.RUnlock()
	schema, ok := registry.schemas[name]
	if !ok {
		return nil, fmt.Errorf("unregistered schema %q", name)
	}
	return schema, nil
}
