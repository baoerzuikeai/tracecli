# tracecli 项目约定

通用 USB 多协议调试工具箱（TUI）。目标：Windows 稳定运行，支持多协议（I2C/SPI/UART/GPIO），协议可插拔扩展。

## 构建与验证

- 构建: `go build ./...`
- 测试: `go test ./... -race`
- Lint: `golangci-lint run`（未安装则跳过，但必须报告）
- **任何任务完成后必须运行构建和测试，并汇报结果，不允许只口头说"写完了"**

## 目录职责

- `cmd/tracecli`: 入口，解析参数 → 起 app → 挂 TUI
- `internal/adapter`: 物理通道，实现 Adapter 接口（specs/02）
- `internal/protocol`: 协议层，实现 Debugger 接口 + 注册表（specs/03）
- `internal/device`: devmap YAML 加载、设备指纹匹配
- `internal/app`: 状态机、事件总线、会话、自动重连
- `internal/script`: 脚本引擎（v0.5 才需要）
- `internal/tui`: bubbletea 视图
- `devmaps/`: 设备寄存器定义库（.yaml，数据驱动，新增设备禁止写 Go 代码）
- `specs/`: 架构与接口契约文档，**只读，改接口需用户批准**

## 代码风格

- 不写注释，除非必要
- 错误必须向上传递，不吞 error
- 单测用 table 风格，放同包 `*_test.go`
- 接口签名以 specs/ 为准，实现不许改接口

## 工作流

- 任务开始前先读 specs/ 相关文档
- 不允许修改 specs/ 中的接口定义；有异议先报告再改
- 不提交 git，等用户验收
- 新增协议走注册表（`protocol.Register`），禁止改动 UI 层和通用 Target 结构
- Capabilities 只能加位，不能改/删已有位（向后兼容）
