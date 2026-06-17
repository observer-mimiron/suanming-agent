# RAG 系统生产级设计指南 — 2025-2026

> RAG 已不再是简单的"检索然后生成"——它是需要精心架构、可观测性和治理的**知识运行时**。

## 1. 架构核心：离线与在线管线分离

生产环境最大错误是把索引和查询混在一个单体里。

| 维度 | 数据管线（离线） | 查询管线（在线） |
|------|-----------------|-----------------|
| 触发 | 定时/事件/批量 | 用户查询，同步 |
| 延迟 | 分钟到小时 | 毫秒到秒 |
| 步骤 | 摄入→分块→嵌入→索引 | 查询→检索→重排→增强→生成 |
| 容错 | 重试可接受 | 低容忍，快速降级 |

## 2. 混合检索是生产基线（非可选）

纯向量搜索有数学上可证明的限制。**2025-2026 年所有权威来源收敛到混合检索：**

1. **稠密向量搜索**（语义相似）+ **稀疏 BM25**（精确关键词）并行运行
2. 用 **RRF（Reciprocal Rank Fusion）** 融合结果
3. 用 **cross-encoder reranker**（Cohere Rerank 3.5 / BGE Reranker / Voyage rerank-2.5）最终排序

**实测影响：**
- 复杂长尾查询 MRR 提升 15-20%（Google Research）
- 结合上下文检索，错误率降低约 69%

## 3. 分块策略决定检索质量

| 策略 | 适用 | 证据 |
|------|------|------|
| **结构感知分块** | API 文档、Markdown | 按标题/端点/章节边界分割 |
| **语义分块** | 叙述内容、法律文档 | 用 LLM 检测主题转移 |
| **层级分块** | 长报告、学术论文 | 句子→段落→章节递归，准确率提升 12%（Microsoft） |
| **Late Chunking** | 需要广泛上下文的文档 | 先 tokenize 整个文档，再分块 |
| **父文档检索** | 精度+上下文平衡 | 索引小块，查询时返回完整父文档 |

**实用规则：**
- 块大小对齐查询粒度
- 叙述内容 10-20% token 重叠；刚性结构禁用重叠
- 考虑**多索引**（政策/FAQ/技术文档分离）——持续优于单一大索引

## 4. 语义缓存：隐藏的成本杠杆

识别语义相似的已回复查询，避免重复 LLM 调用：
- 成本降低高达 **68.8%**
- 响应快达 **65 倍**（缓存命中 <100ms vs LLM 调用数秒）
- 实现：查询 embedding → 向量相似度搜索缓存（阈值 0.85-0.95）

## 5. Agentic RAG：新前沿

简单线性 RAG 只能处理直接问答。复杂任务需要 agent 决定**何时检索、检索什么、是否足够**——迭代直到质量达标。

架构模式：路由 agent → 领域 agent → 专业 RAG 工具（每个封装特定知识源）

**关键缓解措施：** 并行检索、动态模型路由（小模型简单查询/大模型复杂查询）、显式状态管理

## 6. RAGOps：可观测性与评估

约 70% 的系统仍缺乏评估。生产级最低指标：

- **每阶段延迟**（嵌入/检索/重排/生成）：p50, p95, p99
- **检索质量**：precision@k, recall@k, MRR
- **生成质量**：groundedness（每项声明是否有检索上下文支撑）、忠实度、完整性
- **缓存命中率**、**每次查询成本**
- **幻觉率**、**用户满意度**

评估框架：RAGAS、DeepEval；领域系统应构建自己的评估（确定性检查 + 人工验证 ground truth）

## 7. 数据新鲜度

- **全量重建索引**（夜间批量）— 简单但延迟高
- **CDC 增量更新** — 实时但复杂
- **分层知识库**：核心静态层（版本化）+ 频繁更新层（变更日志）+ 按需实时层（API 即时获取）

## 8. 安全与治理

- 多租户隔离（每租户索引/命名空间）
- 权限分段向量 + 基于属性的访问控制
- 查询时访问检查 + prompt 层护栏
- 输入过滤（检索前检测/阻止超出范围的查询）
- 生成后验证（规则引擎或二次 LLM 检查）

## 9. 参考技术栈

| 成熟度 | 向量库 | 重排器 | 可观测性 | LLM 服务 |
|--------|--------|--------|----------|----------|
| 本地开发 | Chroma | 无 | 日志 | Ollama |
| 中型生产 | Qdrant/Weaviate | Cohere Rerank API | Langfuse/Arize | API (OpenAI/Anthropic) |
| 企业高规模 | Milvus 分布式 | 自托管 BGE Reranker | OpenLIT+OTel | vLLM 自托管 |

## 十条金科玉律

1. 把检索当作**一等系统**，有自己的 SLO、指标和预算
2. 从第一天就**混合检索**
3. 尽早实现**语义缓存**——单一最高杠杆的成本优化
4. 将**评估构建进管线**——离线标注数据集 + 在线监控
5. 设计为**嵌入模型可替换**——蓝绿双索引无缝切换
6. **数据管线与查询管线分离**
7. **护栏是 RAG 的一部分**，非事后添加
8. 块大小对齐查询粒度
9. 对复杂任务使用 **Agentic RAG**
10. 监控检索质量漂移——触发自动重建索引

## 权威资源

- [RAG at Scale (Redis, 2026.01)](https://redis.io/blog/rag-at-scale/)
- [Awesome RAG Production (GitHub)](https://github.com/Yigtwxx/Awesome-RAG-Production)
- [AI System Design Guide (GitHub)](https://github.com/ombharatiya/ai-system-design-guide)
- [RAG Pipeline Orchestration (Kestra, 2026)](https://kestra.io/resources/ai/rag-pipeline)
- [RAG Best Practices (Redwerk, 2026.02)](https://redwerk.com/blog/rag-best-practices/)
- [Agentic RAG: Production Architecture (GreenNode, 2026)](https://greennode.ai/blog/rag-ai-agents-low-latency-architecture)
- [Mastering RAG (Apress, 2026, 820页)](https://link.springer.com/book/10.1007/979-8-8688-1808-0)
