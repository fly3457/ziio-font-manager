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

### 字体库管理体验

- 增加用户字体库删除入口：左侧用户字体库可从 Ziio 移除索引记录，删除前明确提示源文件夹和字体文件会保留。
- 收紧 `LibraryService.RemoveRoot`：系统字库禁止删除，扫描中的字体库禁止删除，普通用户字体库继续由 SQLite 级联清理索引数据。
- 补充删除回归测试，覆盖用户字体库移除后的索引清理、系统字库保护和扫描中保护。

### 大字体库扫描稳定性

- 定位 `D:\MyFonts\我的字体\字体全-美的\AMEB___.TTF` 会触发底层字体解析 panic；扫描器现在会把 panic 转为单文件错误，继续扫描后续字体。
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
