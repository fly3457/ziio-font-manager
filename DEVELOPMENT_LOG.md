# Ziio Font Manager 开发日志

本日志记录项目进入 Beta 后的重要功能、较大改进、架构调整和流程变更。小修、小优化、文案微调和对刚完成工作的补丁可以不记录。

## 2026-07-03

### 文档与协作流程

- 更新 `README.md`：从 Wails 模板说明改为项目说明，补充当前功能、技术栈、项目结构、开发命令、Beta 重点和限制。
- 更新 `PROJECT_PLAN.md`：整理为 Beta / v0.3 planning 开发计划，补充当前代码分析、已完成能力摘要、Beta 路线图、测试计划和风险。
- 新增 `AGENTS.md`：记录 Codex 项目级协作规则、Wails + Go + React + TypeScript 工作提示、常用命令和开发日志更新要求。
- 新增 `DEVELOPMENT_LOG.md`：作为根目录开发日志，后续新功能和较大改进需要同步维护。
- 品牌 / 应用名确定为 `Ziio`，完整名为 `Ziio Font Manager`；同步更新窗口标题、前端品牌区、HTML title、Wails 输出名、文档标题和项目说明。
- 将 `README.md` 改为英文完整说明在前、中文完整说明在后的双语项目文档，方便 GitHub 访客快速了解项目，同时保留完整中文上下文。
- 应用数据目录正式切换为 Ziio 命名空间；`README.md` 移除历史数据位置说明，避免刚开始发布阶段留下迁移负担。

影响范围：

- 文档、前端可见品牌、Wails 应用标题 / 输出名和少量后端可见字符串。
- 持久化数据目录为 `%APPDATA%\Ziio\FontManager`，缓存目录为 `%LOCALAPPDATA%\Ziio\FontManager\cache`。
- README 文档面向 GitHub 访客调整为英语优先，并保留完整中文版本。

验证：

- 已重新读取 `README.md` 和 `PROJECT_PLAN.md`，确认 UTF-8 中文正常、无 Wails 模板残留。
- `go test ./...` 通过。
- `npm run build` 通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- README 双语化后已重新读取确认 UTF-8 内容正常；本次文档-only 调整未重新运行测试或构建。
- 数据命名空间切换后重新运行 `go test ./...`，确认后端测试通过。
