# Ziio Font Manager 开发日志

本日志记录项目进入 Beta 后的重要功能、较大改进、架构调整和流程变更。小修、小优化、文案微调和对刚完成工作的补丁可以不记录。

## 2026-08-31

### 开源准备（文档与发布流程）

- 新增 MIT 许可证，以 GitHub 账号 `fly3457` 标注项目版权；前端 package 与 lockfile 同步补充许可证元数据，保留 `private: true` 防止误发布 npm 包。
- README 的中英文部分补充开源说明、Beta 下载入口、从零准备开发环境的步骤和贡献入口，说明 Go 测试前需要生成前端嵌入资源。
- 新增贡献指南和第三方许可说明，收录随附字体、Wails 运行时、当前 Windows 发行文件中 31 个 Go 依赖模块和前端 12 个非开发依赖包等许可正文，明确用户字体不受项目 MIT 许可证覆盖；构建说明补充分发许可文件的要求。
- 删除 `go.mod` 中已注释的本机路径示例，将历史日志中的本地字体完整路径改为文件名描述；不重写 Git 提交历史。

影响范围：许可证、文档和前端包元数据；不改变应用版本、依赖版本、运行逻辑或用户数据。

验证：公开前使用 Gitleaks 8.30.1 扫描既有全部 9 个提交，未发现密钥泄露；本地 v0.2.0 可执行文件的 SHA-256 与 GitHub 已有发行文件一致。重新读取变更文档并检查 UTF-8、链接、package/lockfile 一致性及 `git diff --check`。本次仅涉及文档、许可证、元数据和注释，未运行 `go test`、`npm run build` 或 `wails build`。

## 2026-07-10

### 0.2.0 版本整理

- 应用版本同步到 `0.2.0`，覆盖 `App.GetAppInfo()`、前端设置弹窗 fallback、前端 package 元数据和项目文档。
- `README.md`、`PROJECT_PLAN.md`、`AGENTS.md` 补充当前 0.2.0 能力总结，重点记录 TypeScript 5.9 升级和 `i18next` / `react-i18next` 中英文 i18n。
- 明确后续新增前端可见文案应进入 `frontend/src/i18n.ts`，版本更新需要同步代码、文档和构建说明。

验证：

- 本次整理完成后重新运行 `npm run build`、`go test ./...` 和 `wails build -webview2 embed`。

### SVG 品牌资源与 Windows 图标

- 使用优化后的 `design/SVG/logo.svg` 替换界面左上角原有文字标记，前端通过 `frontend/src/assets/images/logo.svg` 引用品牌 logo。
- 使用同一个透明底 `design/SVG/logo.svg` 重新生成 `build/appicon.png` 与 `build/windows/icon.ico`，ICO 包含 `16, 24, 32, 48, 64, 128, 256` 多尺寸资源。
- 重新执行 Wails 生产构建，生成界面 logo 与 Windows 应用图标统一后的 `build/bin/ZiioFontManager.exe`。

影响范围：

- 前端品牌区视觉资源、Wails Windows 编译图标资源。
- 不新增 Wails API，不改变后端模型、数据库、扫描或安装逻辑。

验证：

- `npm run build` 通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- `wails build -webview2 embed` 通过，生成 `build/bin/ZiioFontManager.exe`。

### 设置弹窗与中英文 i18n

- 前端 TypeScript 升级到 `^5.9.3`，新增 `i18next` 和 `react-i18next`，保持 React 18 与 Vite 3 不变。
- 左上角品牌区移除界面 logo 和 `Ziio Font Manager` 标题，仅保留“本地字体库管理”，右侧新增齿轮设置入口。
- 新增设置弹窗，显示 logo、软件名称、版本号和语言选择；语言偏好通过 `localStorage` 保存。
- 当前主要可见 UI 文案、按钮、菜单、提示、空状态、详情栏、确认框、title / aria / placeholder 已迁入中英文资源。

影响范围：

- 前端依赖、入口初始化、工作台文案、设置弹窗和本地语言偏好。
- 不新增 Wails API，不改变 Go 后端模型、数据库、扫描或安装逻辑；后端原始错误信息仍按原文展示。

验证：

- `npm run build` 通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- `wails build -webview2 embed` 通过，生成 `build/bin/ZiioFontManager.exe`。

### 左侧折叠与网格列数菜单修复

- 修复用户字体库根行点击后无法再次收起的问题，已选中 root 可在展开 / 收起之间切换。
- 保持所有层级文件夹点击名称或展开符均可展开 / 收起，并收紧文件夹行右侧 `...` 菜单按钮对齐。
- 网格列数菜单改为状态控制，hover 网格图标时稳定显示，点击网格图标只切换布局并关闭菜单。

验证：

- `npm run build` 通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- `wails build -webview2 embed` 通过，生成 `build/bin/ZiioFontManager.exe`。

## 2026-07-04

### 左侧导航与可调布局体验

- 左侧字体库 / 文件夹导航改为更紧凑的行高、间距和缩进，优先展示文件夹名称与数量。
- 字体库和文件夹行内操作收纳到 hover/focus 时出现的 `...` 菜单，菜单内提供同步、移除等上下文操作。
- 三栏工作区改为可拖拽调整左侧导航栏和右侧属性栏宽度，并通过 `localStorage` 持久化用户偏好。

影响范围：

- 前端工作台布局、左侧字体库 / 文件夹导航交互和本地 UI 偏好持久化。
- 不新增 Wails API，不改变字体库索引、扫描或安装后端语义。

验证：

- `npm run build` 通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- `go test ./...` 通过，确认未影响后端回归。

### 字体库同步与文件夹级重扫

- 左侧侧栏新增“全量重新扫描”按钮，可一次性对所有已添加字体库排队同步。
- 用户字体库行新增同步图标按钮，可直接同步单个字体库；文件夹行新增同步图标按钮，可递归同步该文件夹及子文件夹。
- 后端新增 `RescanAllRoots` 和 `RescanFolder`，文件夹级扫描只标记该子树内缺失的字体文件，不影响同一字体库的其他文件夹。
- 扫描结果补充 `missing`、`unchanged`、`scope`、`scopePath`，扫描日志会记录 root / folder 范围和增量统计。
- 修正 `UpsertFontFile` 的新增/更新统计判断，避免曾经解析失败且 hash 为空的文件被误算为新增。

影响范围：

- 前端左侧字体库 / 文件夹导航区、Wails 前端绑定、扫描状态模型、扫描任务表兼容迁移。
- 同步操作仍只维护 Ziio 索引，不删除磁盘源文件；删除或移出范围的字体会标记为 `missing` 并从常规列表隐藏。
- 文件夹同步默认递归处理；扫描中的字体库会禁用同步和删除入口。

验证：

- `go test ./...` 通过。
- `npm run build` 通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- `wails build -webview2 embed` 通过，生成 `build/bin/ZiioFontManager.exe`。

## 2026-07-03

### 文档与协作流程

- 更新 `README.md`：从 Wails 模板说明改为项目说明，补充当前功能、技术栈、项目结构、开发命令、Beta 重点和限制。
- 更新 `PROJECT_PLAN.md`：整理为 Beta / v0.3 planning 开发计划，补充当前代码分析、已完成能力摘要、Beta 路线图、测试计划和风险。
- 新增 `AGENTS.md`：记录 Codex 项目级协作规则、Wails + Go + React + TypeScript 工作提示、常用命令和开发日志更新要求。
- 新增 `DEVELOPMENT_LOG.md`：作为根目录开发日志，后续新功能和较大改进需要同步维护。
- 品牌 / 应用名确定为 `Ziio`，完整名为 `Ziio Font Manager`；同步更新窗口标题、前端品牌区、HTML title、Wails 输出名、文档标题和项目说明。
- 将 `README.md` 改为英文完整说明在前、中文完整说明在后的双语项目文档，方便 GitHub 访客快速了解项目，同时保留完整中文上下文。
- 应用数据目录正式切换为 Ziio 命名空间；`README.md` 移除历史数据位置说明，避免刚开始发布阶段留下迁移负担。

### 字体库管理体验

- 增加用户字体库删除入口：左侧用户字体库可从 Ziio 移除索引记录，删除前明确提示源文件夹和字体文件会保留。
- 收紧 `LibraryService.RemoveRoot`：系统字库禁止删除，扫描中的字体库禁止删除，普通用户字体库继续由 SQLite 级联清理索引数据。
- 补充删除回归测试，覆盖用户字体库移除后的索引清理、系统字库保护和扫描中保护。

### 大字体库扫描稳定性

- 定位本地测试字体 `AMEB___.TTF` 会触发底层字体解析 panic；扫描器现在会把 panic 转为单文件错误，继续扫描后续字体。
- 增加扫描诊断日志：`%APPDATA%\Ziio\FontManager\logs\scan.log` 记录扫描开始、候选数量、当前文件、进度、单文件错误和最终统计；`app.log` 记录应用启动和后台扫描入口。
- `GetAppInfo` 增加 `logDir`；应用启动时会创建 `%APPDATA%\Ziio\FontManager\logs`。
- 现代字体解析返回 `error` 状态时计入扫描失败数，避免坏字体没有体现在扫描结果中。
- 新增用户字体库父子路径冲突检查，避免同时添加父目录和子目录导致同一字体文件归属混乱。

影响范围：

- 文档、前端可见品牌、Wails 应用标题 / 输出名和少量后端可见字符串。
- 持久化数据目录为 `%APPDATA%\Ziio\FontManager`，缓存目录为 `%LOCALAPPDATA%\Ziio\FontManager\cache`。
- 扫描日志目录为 `%APPDATA%\Ziio\FontManager\logs`。
- README 文档面向 GitHub 访客调整为英语优先，并保留完整中文版本。
- 用户字体库可从左侧列表移出 Ziio；系统字库不显示删除入口，扫描中的用户字体库会阻止删除。
- 新增用户字体库不能与已有用户字体库形成父子包含关系；已有历史重叠记录不会自动删除。

验证：

- 已重新读取 `README.md` 和 `PROJECT_PLAN.md`，确认 UTF-8 中文正常、无 Wails 模板残留。
- `go test ./...` 通过。
- `npm run build` 通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- README 双语化后已重新读取确认 UTF-8 内容正常；本次文档-only 调整未重新运行测试或构建。
- 数据命名空间切换后重新运行 `go test ./...`，确认后端测试通过。
- 用户字体库删除功能完成后重新运行 `go test ./...`，通过。
- 用户字体库删除功能完成后重新运行 `npm run build`，通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- 大字体库扫描稳定性改进完成后重新运行 `go test ./...`，通过。
- 大字体库扫描稳定性改进完成后重新运行 `npm run build`，通过；Vite 仍提示第三方模块级 `"use client"` 指令被忽略，为构建警告。
- 大字体库扫描稳定性改进完成后额外运行 `wails build -webview2 embed`，通过，生成 `build/bin/ZiioFontManager.exe`。
