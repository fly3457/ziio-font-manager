# AGENTS.md

本文件是 Ziio Font Manager 仓库给 Codex / AI coding agent 的项目级指导。后续在本仓库内工作时，优先遵守这里的约定；如果用户在当前对话里给出更具体要求，以用户最新要求为准。

## 项目概览

- 项目类型：Windows 11 桌面字体管理工具。
- 品牌名：Ziio。
- 完整应用名：Ziio Font Manager。
- 技术栈：Wails v2.12 + Go 1.25 + React 18 + TypeScript 4.6 + Vite 3。
- 数据库：SQLite，使用 `modernc.org/sqlite` 纯 Go 驱动。
- 字体解析：`github.com/tdewolff/font`。
- Windows 集成：Registry、GDI `AddFontResourceEx` / `RemoveFontResourceEx`、`WM_FONTCHANGE`。
- 当前阶段：已过 MVP，进入 Beta / v0.3 planning。

## 工作原则

- 默认使用中文回复和更新文档，代码标识符、命令、错误信息保持原文。
- 文档和源码统一使用 UTF-8。
- 不要把 `frontend/node_modules`、`frontend/dist`、`build/bin` 当作核心源码处理。
- 修改时优先贴合现有结构：Go 后端在 `internal/*`，React 前端在 `frontend/src/*`，Wails 绑定由 `frontend/wailsjs/*` 承接。
- 对 Windows 字体安装、卸载、注册表和管理员权限相关代码保持谨慎；不得绕过系统字体保护逻辑。
- 只做用户要求范围内的修改，不顺手重构无关模块。

## 开发日志要求

根目录维护开发日志文件：`DEVELOPMENT_LOG.md`。

- 每次新增功能、较大体验改进、架构调整、扫描/安装/预览等核心流程变化，都需要同步更新开发日志。
- 如果只是对刚刚完成的功能做小调整、小优化、文案修正、样式微调或缺陷补丁，可以不更新开发日志。
- 日志应记录日期、变更类型、摘要、影响范围和验证情况。
- 如果本次工作明确没有运行测试或构建，也要在日志条目里写清楚原因。

## 常用命令

后端测试：

```powershell
go test ./...
```

前端构建检查：

```powershell
Set-Location frontend
npm run build
```

Wails 开发模式：

```powershell
wails dev
```

生产构建：

```powershell
wails build -webview2 embed
```

只改文档时，不需要运行 `go test`、`npm run build` 或 `wails build`，但需要重新读取相关文档确认 UTF-8 中文正常。

## Wails + Go 最佳实践提示

处理后端功能时，可以参考这些工作提示：

- 先确认 Wails 暴露给前端的服务边界：`LibraryService`、`FontService`、`InstallService` 是否已经覆盖需求。
- 新增前后端交互时，优先补充 `internal/models` 的结构体，再通过 Wails 生成或维护前端类型映射。
- 数据需要持久化时，优先走 `internal/store`，保持 SQLite schema、迁移和查询集中管理。
- 扫描字体库、生成预览、安装字体这类耗时或高风险操作要有进度、错误状态和可恢复策略。
- Windows 字体安装逻辑必须区分 `user` 和 `machine` 范围；`machine` 操作需要管理员权限。
- 当前版本为保留既有索引，预览缓存和持久数据仍暂时沿用历史 `Yuncii\FontManager` 数据目录；如需迁移到 Ziio 命名空间，必须提供自动迁移或显式导入流程。

## React + TypeScript 最佳实践提示

处理前端功能时，可以参考这些工作提示：

- 保持三栏工作台结构：左侧字体库/文件夹导航，中间字体结果区，右侧详情面板。
- 状态变化要考虑搜索、筛选、分页、选中项、批量勾选、预览队列之间的联动。
- 字体列表默认分页大小是 20；触底加载逻辑不要误显示数据库总数。
- 预览加载应继续保持懒加载和低并发，避免大字体库一次性生成大量预览。
- UI 文案应尽量说明操作结果和失败原因，尤其是权限不足、旧格式受限、坏字体、缺失文件。
- 长路径、长字体名、超长样本文字要避免撑坏布局。

## 可直接使用的任务提示模板

新增后端能力：

```text
请在现有 Wails + Go 服务边界内实现这个后端能力：<能力描述>。
要求先检查 internal/library、internal/store、internal/models 是否已有可复用结构；
如涉及持久化，补齐 SQLite schema/迁移和查询；
如影响前端调用，同步更新 TypeScript 类型和 API 封装；
完成后运行 go test ./...，并更新 DEVELOPMENT_LOG.md。
```

新增前端体验：

```text
请在现有 React + TypeScript 三栏工作台结构内实现这个前端体验：<体验描述>。
要求保持当前视觉风格，处理加载、空状态、错误状态和长文本；
如需要后端数据，先复用现有 Wails API，不够再补后端；
完成后运行 npm run build，并在属于较大改进时更新 DEVELOPMENT_LOG.md。
```

修改扫描/预览流程：

```text
请优化字体扫描/预览流程：<问题描述>。
要求考虑大字体库、坏字体、超时、缓存、错误可见性和界面响应；
不能让单个异常文件阻塞整个任务；
完成后运行 go test ./...，如涉及前端也运行 npm run build，并更新 DEVELOPMENT_LOG.md。
```

修改安装/卸载流程：

```text
请调整 Windows 字体安装/卸载流程：<需求描述>。
要求明确 user/machine 范围、管理员权限、注册表写入、系统字体保护、失败提示和操作日志；
不得默认删除用户源文件；
完成后运行 go test ./...，并更新 DEVELOPMENT_LOG.md。
```

只改文档：

```text
请只更新项目文档：<文档目标>。
不要修改源码，不运行构建；
完成后重新读取相关 Markdown，确认 UTF-8 中文正常、内容无模板残留；
如果属于阶段性文档或流程变更，更新 DEVELOPMENT_LOG.md。
```
