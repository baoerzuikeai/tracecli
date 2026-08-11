# 架构文档

## 总览（六层）

```
┌────────────────────────────────────────────────┐
│ internal/tui/    UI 层：bubbletea 视图          │
├────────────────────────────────────────────────┤
│ internal/app/    应用层：状态机、事件总线、重连  │
├────────────────────────────────────────────────┤
│ internal/protocol/ 协议层：Debugger + 注册表     │
├────────────────────────────────────────────────┤
│ internal/adapter/  适配器层：CH341/FTDI VCP     │
├────────────────────────────────────────────────┤
│ internal/device/   设备库：YAML 加载、指纹匹配   │
├────────────────────────────────────────────────┤
│ internal/script/   脚本引擎（v0.5）              │
└────────────────────────────────────────────────┘
```

## 目录结构

```
tracecli/
├── cmd/tracecli/main.go
├── internal/
│   ├── adapter/          # Adapter 接口 + 实现（每芯片一个文件）
│   ├── protocol/         # Debugger 接口 + 注册表 + i2c/spi/uart
│   ├── device/           # devmap 加载、指纹匹配
│   ├── app/              # 状态机、事件总线、会话、重连
│   ├── script/           # 脚本引擎
│   └── tui/              # bubbletea 视图模型
├── devmaps/              # 设备寄存器定义库（.yaml）
├── specs/                # 本文档族（只读）
└── config.yaml           # 默认串口、快捷键、波特率
```

## 核心设计一：事件驱动（UI 不阻塞）

```
串口 goroutine（常驻读）→ 字节流 → 协议层解码 → 事件总线 → TUI 订阅刷新
         ↓
掉线/数据到达均为事件。UI 只做纯渲染，绝不直接碰串口 IO。
```

## 核心设计二：连接状态机（Windows 稳定性关键）

```
Disconnected ──open──> Connecting ──成功──> Ready
      ↑                    │失败              │设备拔出/超时
      └───── 自动重连(指数退避) ◄──────────────┘ Error
```

- Ready 下检测掉线 → 自动重连 → 重扫 → 恢复断线前的焦点层级
- COM 端口号变化也要能跟上（Windows WMI 事件订阅）

## 核心设计三：自动识别管线

1. `adapter.ID()` → 判断芯片类型（CH341/FTDI）→ 加载对应命令集
2. 按当前协议 `Probe` → 扫描地址收 ACK → 在线设备列表
3. 读指纹寄存器 → 匹配 `devmaps/` 已知设备
4. 命中 → 挂载 devmap 寄存器树；未命中 → `Unknown@0xXX`

## 数据流示例（读寄存器）

```
键盘按 r
→ app.Handler 校验会话
→ protocol.I2C.Read(0x50, 0x10, 4)
→ adapter.CH341 拼命令包 → 串口 → 芯片 → I2C 总线
→ 返回字节 → 解码 → 事件 {target, addr, data}
→ TUI 表格刷新该行
```

## 扩展约束（所有实现必须遵守）

- 新增协议禁止修改：registry、UI 层、通用 Target 结构
- 协议特有概念（如 SPI 的 CPOL/CPHA）不得泄漏进通用 Target
- Capabilities 只能加位，不能改/删已有位
- 新增设备 = 加一个 `devmaps/*.yaml`，禁止写 Go 代码
