# Skill 管理原理

> 参考 [OpenHarness `src/openharness/skills](https://github.com/HKUDS/OpenHarness/tree/main/src/openharness/skills)`

Skill 是一段预写好的 Markdown 指令，用于在特定任务场景（如 `plan`、`debug`、`commit`）下为 LLM 注入"操作手册"。每个 skill 由 `name + description + content` 组成：索引（name/description）在系统提示里常驻，正文（content）只有当 agent 主动调用 `skill` 工具时才按需加载，从而在"提示词膨胀"和"能力可发现性"之间取得平衡。

## 1. 数据模型

对应 `skills/types.go`：

```go
type SkillDefinition struct {
    Name        string // 唯一标识
    Description string // 常驻在系统提示里，供 LLM 判断是否调用
    Content     string // 正文（Markdown），按需加载
    Source      string // bundled | user
    Path        string // 物理位置（embed 虚拟路径或磁盘路径）
}
```

## 2. 文件布局

Skill 按"每个目录一个 skill"的约定组织，便于配套放脚本、示例、素材。

### 2.1 内置（bundled）

通过 `//go:embed content/*.md` 打包进二进制，扁平布局：

```
skills/
├── bundled.go          # go:embed 入口
└── content/
    ├── plan.md
    ├── debug.md
    ├── commit.md
    ├── review.md
    ├── simplify.md
    ├── diagnose.md
    └── test.md
```

### 2.2 用户（user / extra）

磁盘目录，采用 `<skill_name>/SKILL.md` 约定。`<skill_name>` 即兜底的 skill 名：

```
<config_dir>/skills/
├── my-skill/
│   └── SKILL.md
└── other-skill/
    └── SKILL.md
```

`<config_dir>` 由 `config.GetConfigDir()` 决定（`$XDG_CONFIG_HOME/openharness` 或 `~/.config/openharness`），不存在时自动创建。

### 2.3 Markdown 头部约定

推荐使用 YAML frontmatter 显式声明元数据：

```md
---
name: plan
description: Design an implementation plan before coding.
---

# plan

...正文...
```

若没有 frontmatter，则回退到"首个 `# 标题` 作为 name、首段非空非标题文字作为 description（截断到 200 字符）"。

## 3. 加载流程

入口：`skills.LoadSkillRegistry(opts...)`（见 `skills/loader.go`）。

```
LoadSkillRegistry
 ├─ NewSkillRegistry()                    // 进程级单例
 ├─ GetBundledSkills()                    // 1. 内置
 ├─ LoadUserSkills()                      // 2. ~/.config/openharness/skills
 ├─ LoadSkillsFromDirs(extraSkillDirs)    // 3. 额外目录（CLI/settings）
 └─ (TODO) 插件目录下的 skills            // 4. 按 cwd 发现的 plugin skills
```

注册顺序决定覆盖关系：**后注册的同名 skill 覆盖前者**，即 `plugin > extra > user > bundled`，允许用户无痛 override 内置能力。

## 4. 注入到系统提示

由 `prompts.buildSkillSection` 组装（见 `prompts/context.go`）：

```md
# Available Skills

The following skills are available via the `skill` tool.
When a user's request matches a skill, invoke it with `skill(name="<skill_name>")`
to load detailed instructions before proceeding.

- **plan**: Design an implementation plan before coding.
- **debug**: ...
- ...
```

注意只注入了 `name + description`，**正文 content 不进系统提示**。LLM 根据索引判断是否需要某个 skill，再通过 `skill` 工具按需取用，典型的"lazy loading"，避免所有 skill 正文同时灌入上下文。

## 5. 与 Memory 的对比


| 维度   | Skill                  | Memory                           |
| ---- | ---------------------- | -------------------------------- |
| 粒度   | 预写好的操作手册（开发者/产品提供）     | 用户/项目积累的知识（会话中动态产出）              |
| 存储   | 内置 + 用户目录 + 额外目录       | 每项目一个哈希目录                        |
| 作用域  | 全局可复用                  | 项目隔离                             |
| 注入内容 | name + description（索引） | MEMORY.md 索引 + 相关性打分命中的 topic    |
| 检索   | 按 name 精确调用            | 轻量关键词打分（title×2 + description×1） |
| 写入时机 | 版本发布 / 手动维护            | /memory 命令或后台 agent              |


两者都走"在系统提示里只放索引，正文按需加载"的思路，只是 skill 的入口是工具调用，memory 的入口是相关性检索。

## 6. 设计要点总结

1. **两层来源 + 覆盖语义**：bundled 保证开箱即用，user/extra/plugin 允许覆盖；`plugin > extra > user > bundled`。
2. **目录式组织**：`<skill>/SKILL.md` 约定让一个 skill 能带附属文件（脚本、示例），未来扩展空间大。
3. **frontmatter 优先、启发式兜底**：不强制 YAML，避免引入解析依赖；缺失字段可从标题/首段自动推断。
4. **稳定排序 + 去重**：文件名排序 + `seen` 集合 + map 覆盖，保证同一输入永远得到同一索引。
5. **索引常驻、正文按需**：只把 description 注入系统提示，正文等 LLM 调用 `skill` 工具时再取，控制 token 成本。
6. **Registry 单例**：进程内复用，无需每次请求扫盘。

