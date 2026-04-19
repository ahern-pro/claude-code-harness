package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/li-zeyuan/claude-code-harness/config"
)

/* Memory 索引文件：
如：
# Memory Index
- [Coding style](coding_style.md)
- [API keys location](api_keys_location.md)
- [Deploy process](deploy_process.md)

NOTE：核心用途是将 Memory Index 作为系统提示词的一部分(LoadMemoryPrompt)，让 LLM 在推理时能"回忆"起项目记忆。以便按需拉取对应的 .md记忆文件。
	而不是在 搜索相关记忆文件（FindRelevantMemories） 阶段作为索引文件。
*/
const memoryEntrypointFileName = "MEMORY.md"

func GetProjectMemoryDir(cwd string) (string, error) {
	if cwd == "" {
		return "", fmt.Errorf("cwd must not be empty")
	}

	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path for %q: %w", cwd, err)
	}
	// Match Python's Path.resolve() semantics by also resolving symlinks when
	// the target exists. Fall back to the absolute path otherwise so callers
	// can key memory off a not-yet-created directory.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}

	sum := sha1.Sum([]byte(abs))
	digest := hex.EncodeToString(sum[:])[:12]

	dataDir, err := config.GetDataDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}

	memoryDir := filepath.Join(dataDir, "memory", fmt.Sprintf("%s-%s", filepath.Base(abs), digest))
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		return "", fmt.Errorf("create memory dir %q: %w", memoryDir, err)
	}
	return memoryDir, nil
}

func GetMemoryEntrypoint(cwd string) (string, error) {
	dir, err := GetProjectMemoryDir(cwd)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, memoryEntrypointFileName), nil
}
