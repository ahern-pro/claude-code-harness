# Memory 管理原理

> 参考 [OpenHarness `src/openharness/memory`](https://github.com/HKUDS/OpenHarness/tree/508937c996877f7b4e198610fb6d38490e0eb8b5/src/openharness/memory)

Memory 模块用来在多次会话之间，为同一个项目持久化 "用户/项目上下文"。它不是向量库，而是一个纯本地的 Markdown 文件集合，配以简单的索引和启发式检索。

## 1. 文件布局

```
<data_dir>/memory/<project>-<sha1_12>/
├── MEMORY.md          # 索引，列出每条 memory 的标题和文件名
├── topic_a.md         # 每条 memory 一个 .md 文件
├── topic_b.md
└── ...
```

每个 topic 文件可选 YAML frontmatter 提供元数据：

```md
---
name: 部署流程
description: 生产环境发布步骤
type: runbook
---
正文……
```

## 2. 增删改查（`manager.go`）

- 增/改/删都会同步修改 MEMORY.md 文件。
- 更新时机：
   - 用户在交互会话里执行 /memory add|remove;
   - 后台子 agent 定期校验和更新（待实现）


## 3. 注入到系统提示-为 LLM 提供索引信息

> 涉及文件：topic_xx.md

`LoadMemoryPrompt` 生成一段 Markdown，作为系统提示的一部分：

- 固定说明：persistent memory 目录位置、用途（跨会话知识）、写作建议（简短主题文件 + `MEMORY.md` 索引）。
- 附带 `MEMORY.md` 最多 200 行的内容（用 ``` ```md ``` 代码块包裹）；若尚未创建则提示 `(not created yet)`。

这样 agent 每次启动都能看到索引，决定是否需要通过工具读取具体 topic 文件或追加新条目。


## 4. 注入到系统提示-相关性 memory 检索

> 涉及文件：MEMORY.md

`FindRelevantMemories` 实现一个轻量打分算法：

1. 分词：
   - ASCII：`[A-Za-z0-9_]+` 且长度 ≥ 3；
   - CJK：每个汉字（Unicode `\u4e00-\u9fff`、`\u3400-\u4dbf`）作为独立 token。
2. 对 `ScanMemoryF` 的结果逐条打分：
   - 命中 `title`，score * 2
   - 命中 `description`, score * 1；
3. 按 `(-score, -modified_at)` 排序，取前 `max_results` 条。

没有向量、没有 embedding，优点是零依赖、可解释；缺点是对同义词不敏感，所以 `description` 和 `title` 的质量直接决定召回效果。


## 5. 设计要点总结

1. **按项目哈希目录**：天然隔离，不依赖项目内文件，源码 clean 也不会丢记忆。
2. **纯 Markdown + 索引**：人类可读、可手工编辑，便于 diff/备份。
3. **Frontmatter 驱动检索**：鼓励结构化元数据，`title`/`description` 权重 2 倍。
4. **启发式而非语义检索**：低成本、确定性强，适合 agent 级别的 "提醒式" 记忆。
5. **提示注入有上限**：索引截取前 200 行，避免 MEMORY 膨胀撑爆 prompt。

## 遗留问题
如何做 memory 更新

