# 扩展路线图（specs/05）

实现顺序即依赖顺序。每个版本结束必须全部测试通过。

## 版本规划

| 版本 | 内容 | 依赖 | 验收 |
|------|------|------|------|
| v0.1 | I2C + CH341 VCP + 设备树浏览 + 自动识别 + 重置 + Windows 稳定运行 | 无 | 真机（CH341 + EEPROM）读写、掉线重连 |
| v0.2 | 协议注册表 + 能力矩阵落地（此版**按扩展接口写**，不为快而省）| v0.1 | 内置 i2c 走注册表加载 |
| v0.3 | UART 透传（验证第二协议走通全链路，UI 零改动）| v0.2 | 串口透传 loopback 测试 |
| v0.4 | SPI（引入 CapWaveform 波形面板）| v0.3 | SPI flash 读 ID |
| v0.5 | 脚本引擎 + 操作序列回放（消费 devmap operations）| v0.2 | 录制→回放全流程 |
| v1.0 | 插件目录规范（第三方协议放 `.tracecli/protocols/`）、打包分发 | v0.4 | 独立协议插件加载 |

## 扩展规则（v0.1 就必须遵守，agent 禁止绕过）

1. **接口通用化先行**：v0.1 写 I2C 时接口就是 `Target + Capabilities` 的通用形态，不为 I2C 特化
2. **新增协议禁止修改**：registry、UI 层、通用 Target 结构、事件类型
3. **Capabilities 只加位**，改/删已有位即破坏兼容
4. **新增设备禁止写 Go 代码**，只加 yaml
5. 协议特有参数进各自 Options + config.yaml 分节，不进通用接口

## 命名规范

- 模块路径：`github.com/baoerzuikeai/tracecli`
- 适配器实现文件：`ch341_vcp.go`、`ftdi_vcp.go`、`mock.go`
- 协议实现文件：`i2c.go`、`spi.go`、`uart.go`
- devmap 文件名即设备键：`mpu6050.yaml`

## 已知风险

| 风险 | 缓解 |
|------|------|
| CH341 VCP 模式命令格式随芯片版本差异 | 单测锁定字节格式，真机联调阶段验证 |
| Windows conhost 终端兼容 | 检测 `WT_SESSION`，不在 Windows Terminal 则自动拉起 `wt.exe` |
| USB 延迟计时器导致读写慢 | 配置 Latency 字段，Windows 注册表调优文档化 |
