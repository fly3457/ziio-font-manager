# Ziio Font Manager

Ziio Font Manager 是一个面向 Windows 11 的本地字体管理工具。项目已经完成 MVP 验证，当前进入 Beta / v0.3 planning 阶段，重点从“核心功能可用”转向“稳定、清晰、可发布、可维护”。

应用使用 Wails 将 Go 后端能力和 React 前端界面打包为桌面程序，主要解决本地字体库索引、预览、检索、收藏、安装和卸载管理问题。

## 当前功能

- 字体库管理：添加用户字体目录，扫描 Windows 系统字体目录，并按“用户字体库 / 系统字库”分区展示。
- 后台索引：添加字体库、重新扫描和扫描系统字库都会启动后台任务，避免大目录阻塞界面。
- 字体检索：支持按 family、样式、文件名、路径搜索，并支持收藏、已安装状态筛选。
- 文件夹浏览：左侧字体库下内嵌文件夹树，支持按层级展开、收起和本地持久化展开状态。
- 字体列表：支持网格 / 列表视图、卡片列数切换、20 条分页和触底自动加载。
- 字体预览：按可见字体懒加载预览资源，支持自定义样本文字和字号，预览失败时降级显示。
- 字体详情：展示格式、文件大小、字形数量、版本、厂商、PostScript 名称、安装记录和错误状态。
- 安装管理：支持当前用户普通安装、快捷安装、卸载；所有用户范围作为高级能力，需要管理员权限。
- 操作反馈：安装过程通过 Wails 事件上报进度，界面显示成功 / 失败结果和操作提示。

## 技术栈

- 桌面壳：Wails v2.12
- 后端：Go 1.25
- 前端：React 18、TypeScript 4.6、Vite 3
- UI 图标：lucide-react
- 数据库：SQLite，使用纯 Go 驱动 `modernc.org/sqlite`
- 字体解析与预览：`github.com/tdewolff/font`
- Windows 集成：Registry、GDI `AddFontResourceEx` / `RemoveFontResourceEx`、`WM_FONTCHANGE`

## 项目结构

```text
.
├── app.go                       # Wails 应用初始化、服务绑定和预览字体静态处理
├── main.go                      # Wails 启动入口
├── go.mod / go.sum              # Go 依赖
├── wails.json                   # Wails 项目配置
├── internal/
│   ├── appdirs/                 # 应用数据目录和缓存目录解析
│   ├── fontmeta/                # 字体格式识别、元数据解析、预览子集生成
│   ├── library/                 # Wails 暴露的字体库、字体、安装服务
│   ├── models/                  # 前后端共享的数据模型
│   ├── scanner/                 # 字体目录扫描、增量索引、解析超时保护
│   ├── store/                   # SQLite schema、查询、收藏、安装记录、扫描任务
│   └── winfont/                 # Windows 字体安装、卸载、注册表和权限边界
├── frontend/
│   ├── src/                     # React 应用源码
│   ├── wailsjs/                 # Wails 生成的前端绑定
│   └── package.json             # 前端依赖和脚本
└── build/                       # Wails 构建配置和本地产物
```

说明：`frontend/node_modules`、`frontend/dist` 和 `build/bin` 属于依赖或构建产物，不作为核心源码结构的一部分。

## 数据位置

为避免品牌重命名后丢失既有索引，当前版本暂时沿用历史数据命名空间：

- 持久化数据：`%APPDATA%\Yuncii\FontManager`
- SQLite 数据库：`%APPDATA%\Yuncii\FontManager\fontmanager.db`
- 预览缓存：`%LOCALAPPDATA%\Yuncii\FontManager\cache`
- 预览字体文件：`%LOCALAPPDATA%\Yuncii\FontManager\cache\previews`

后续如果实施数据目录迁移，应提供从历史目录到 Ziio 新目录的自动迁移或显式导入流程。

## 开发命令

在项目根目录运行：

```powershell
wails dev
```

后端测试：

```powershell
go test ./...
```

前端构建检查：

```powershell
Set-Location frontend
npm run build
```

生产构建：

```powershell
wails build -webview2 embed
```

构建后的便携版可执行文件通常位于：

```text
build\bin\ZiioFontManager.exe
```

## Beta 重点

Beta 阶段优先关注：

- 大字体库扫描稳定性、可取消 / 可重试能力和错误字体可见化。
- 所有用户安装 / 卸载的权限体验和安全边界。
- 预览缓存治理、预览任务可观测性和旧格式字体降级说明。
- 批量操作、筛选、长路径、空状态和响应式布局打磨。
- 版本号、构建产物、安装包 / 便携包和发布检查清单。

## 当前限制

- 所有用户范围的安装 / 卸载要求应用以管理员身份运行，自动 UAC 提权助手尚未实现。
- WOFF / WOFF2 参与扫描和预览，但默认不作为 Windows 系统字体安装格式。
- Type1、FON、FNT、FOT 等旧格式以管理能力优先，元数据和预览可能受限。
- 首次扫描超大字体库仍需要时间完成索引，但不会阻塞添加字体库操作。
- 单个字体元数据解析超过 12 秒会被标记为错误并跳过，避免拖住整个扫描任务。
