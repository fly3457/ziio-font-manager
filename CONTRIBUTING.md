# 参与 Ziio Font Manager

欢迎通过 [Issues](https://github.com/fly3457/ziio-font-manager/issues) 反馈问题或提出建议，也欢迎提交 Pull Request。

## 开始开发

请先阅读 [README](README.md) 中的环境准备和构建说明。当前支持目标是 Windows 11，项目处于 Beta 阶段；较大的功能或架构调整建议先通过 Issue 讨论。

- Go 后端位于 `internal/*`，React / TypeScript 前端位于 `frontend/src/*`。
- 新增用户可见文案时，同步维护 `frontend/src/i18n.ts` 中的中文和英文资源。
- 字体安装和卸载必须保留系统字体保护，并区分当前用户与所有用户范围；不要默认删除用户源文件。
- 不提交 `frontend/node_modules`、`frontend/dist`、`build/bin`、本地数据库、日志、密钥或无权再分发的字体。

## 提交前检查

在仓库根目录运行以下命令。前端构建需在 Go 测试之前完成，因为入口程序会嵌入 `frontend/dist`：

```powershell
npm --prefix frontend ci
npm --prefix frontend run build
go test ./...
```

涉及桌面集成时，再运行 `wails build -webview2 embed`，并在 Windows 上检查受影响的流程。仅修改文档时可以不运行测试或构建，但应确认 UTF-8 中文和链接正常。

Pull Request 请说明修改原因、用户可见变化和实际验证结果。新增功能、较大体验改进或核心流程变化，还需更新 [开发日志](DEVELOPMENT_LOG.md)。保持修改聚焦，不混入无关重构。

## 反馈问题

请提供应用版本、Windows 版本、复现步骤、预期行为和实际结果。提供日志或截图前，请移除用户名、个人路径和其他敏感信息；字体样本只能使用你有权分享的文件。不要在公开 Issue 或 Pull Request 中上传访问令牌或其他秘密。

项目原创代码使用 [MIT 许可证](LICENSE)，第三方资源保留各自许可，详见 [第三方许可说明](THIRD_PARTY_NOTICES.md)。提交代码或资源前，请确认你有权按适用许可证分享这些内容。
