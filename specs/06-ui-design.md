# TUI 界面设计规范（specs/06）

来源：Gemini 设计稿 v2 + 人工审核修正。实现以本文件为准。

## 1. 界面布局

### 1.1 顶部栏（Header，固定 3 行）
```
┌─ tracecli ────────────────────────────────────────────────┐
│ [1: I2C] [2: SPI] [3: UART]    ● CONNECTED  I2C@400kHz     │
│                                   ADDR:0x50  ERR:0  [?]HELP│
```
- 左侧：协议 Tab（数字键 1-9 切换，当前激活高亮）
- 右侧：连接状态（● 绿=就绪 / 黄=扫描 / 红=错误）+ 当前协议速率 + 错误计数
- 布局自适应：Tab 靠左，状态靠右，不设固定比例

### 1.2 主工作区（自适应高度）两栏

#### ① 设备列表（25%）
- **数据源**：仅显示当前激活协议 Tab 的 Probe 结果（Target 列表）
- 行格式：`0x50  EEPROM AT24C32A`（地址 + 识别名）
- 未知设备：`0x7C  Unknown@0x7C`

#### ② 寄存器表（75%）
- 顶部**分组条**（固定 1 行，横向滚动）：`00-1F / 20-3F / 40-5F ...`，按 `addr >> 5` 自动分组
- 模式行：`[R]读取 [W]写入 [x/b]BIN列:关`
- 列（见第 7 节列宽预算）：`ADDR | HEX | BIN(按需) | FIELD | 描述`
- 底部位域可视化（固定 3 行）：
  ```
  0x6B = 0x00
  [7:0][6:0][5:0][4:0][3:0][2:0][1:0][0:0]
  SLEEP(6) rw  DEVICE_RESET(7) w
  ```

### 1.3 底部仪表行（instrumentBar，固定 2 行）
- 读/写流量 sparkline（▁▂▃▄▅▆▇█ 双柱，记录最近 60 秒）
- 空数据时显示占位提示

### 1.4 底部（固定高度）
- helpPanel（固定 1 行）：常用快捷键条，`?` 展开完整列表
- statusbar（固定 1 行）：`状态:就绪 | 速率:400kHz | 包:1204 OK`（无时钟）

## 2. 布局树

```text
root: JoinVertical
├── header (固定 3 行): JoinHorizontal(自适应)
├── main (自适应): JoinHorizontal
│   ├── devicePanel (25%): bubbles list
│   └── registerPanel (75%)
│       ├── groupBar (固定 1 行): 横向分组 Tab（自定义，背景胶囊）
│       ├── table: bubbles table
│       └── bitFieldDetail (固定 3 行): viewport
├── instrumentBar (固定 2 行): 流量 sparkline（自定义）
├── helpPanel (固定 1 行): bubbles help
└── statusbar (固定 1 行): 自定义 lipgloss
```

## 3. 调色方案（Catppuccin Mocha）

> 统一采用 Catppuccin Mocha 变体（官网 https://catppuccin.com 获取全表）。

### 3.1 四态交互色
| 状态 | HEX | Catppuccin 名 | 用途 |
|------|-----|---------------|------|
| 选中项 | `#CBA6F7` | Mauve | 选中行主色、焦点边框、激活 Tab |
| 选中行背景 | `#313244` | Surface0 | selectedRow 背景 |
| 扫描中 | `#89DCEB` | Sky | spinner、扫描提示 |
| 就绪 | `#A6E3A1` | Green | 连接成功、状态 OK |
| 错误 | `#F38BA8` | Red | 错误信息、断连 |

### 3.2 语义色
- 协议色：I2C Mauve `#CBA6F7` / SPI Blue `#89B4FA` / UART Teal `#94E2D5`
- 背景主 Base `#1E1E2E` / Header Mantle `#181825` / 最深 Crust `#11111B`
- 边框亮 Lavender `#B4BEFE`（焦点）/ 普通边框 Surface1 `#45475A`
- 普通文字 Text `#CDD6F4` / 弱化 Overlay0 `#6C7086`
- 数值 Yellow `#F9E2AF` / 代码 Blue `#89B4FA`
- spinner Sky `#89DCEB`
- 面板装饰 Surface0 `#313244` / Surface1 `#45475A`（胶囊、表头底）
- 表头文字 Teal `#94E2D5`

## 4. 组件样式表（Lip Gloss Style Rules）

> **实现约束**：Lip Gloss 无法设置终端级背景。所有面板/整行背景必须显式 `Width(targetWidth)` + 空格补齐（`lipgloss.PlaceHorizontal`）实现填满。

| 样式名 | 属性 | 说明 |
|--------|------|------|
| root | Background #1E1E2E, Foreground #CDD6F4 | 全局（Base）|
| header | Background #1E1E2E, RoundedBorder, BorderBackground #1E1E2E, Padding(0,1), Height(3) | 标题+Tab+状态（Base）|
| deviceList | Background #1E1E2E, RoundedBorder, BorderForeground #CBA6F7, BorderBackground #1E1E2E, Padding(0,1) | 25%（Base/Mauve）|
| groupBar | Background #1E1E2E, 分隔色 #45475A, Padding(0,2) | 分组条（Base/Surface1）|
| registerTable | Background #1E1E2E, RoundedBorder, BorderForeground #94E2D5, BorderBackground #1E1E2E, Padding(0,2) | 75%（Base/Teal）|
| tableHeader | Foreground #94E2D5, Bold, Underline | 表头 |
| selectedRow | Background #313244, Foreground #CDD6F4, Bold | 选中行（Surface0）|
| bitFieldDetail | Background #1E1E2E, 上方 1 行 Foreground #45475A 分隔线，内容缩进 2 格 | 位域展开区（Base/Surface1）|
| instrumentBar | Background #1E1E2E, NormalBorder, BorderForeground #45475A, BorderBackground #1E1E2E | 流量仪表（Base）|
| helpPanel | Background #1E1E2E, NormalBorder, BorderForeground #45475A, BorderBackground #1E1E2E | 快捷键（Base）|
| statusbar | Background #1E1E2E, Foreground #A6E3A1, NormalBorder, BorderBackground #1E1E2E | 状态栏（Base）|
| spinner | Foreground #89DCEB | 动画（Sky）|
| toastError | Background #1E1E2E, Foreground #F38BA8 | 错误弹窗 |
| toastSuccess | Background #1E1E2E, Foreground #A6E3A1 | 成功弹窗 |

## 5. 组件映射（Bubbles）

| 区域 | 组件 |
|------|------|
| 协议 Tab | 自定义（lipgloss），1-9 切换 |
| 设备列表 | bubbles/list |
| 寄存器组 | bubbles/list |
| 寄存器表 | bubbles/table |
| 位域详情 | bubbles/viewport |
| 扫描动画 | bubbles/spinner |
| 状态栏 | 自定义（lipgloss）|
| 数值输入 | bubbles/textinput（w 弹窗）|
| Toast | 自定义 |

## 6. 快捷键与交互

| 键 | 动作 | 事件流 |
|----|------|--------|
| 1-9 | 切换协议 Tab | Update → 重滤设备列表 |
| x / b | 展开/折叠 BIN 列 | Update → 重算列宽 |
| ↑/↓ | 光标上下 | list/table.CursorUp/Down |
| ←/→ / Tab / Shift+Tab | 切换面板焦点 | FocusNext/Prev |
| Enter | 下钻/读取 | DrillDown |
| r | 刷新寄存器 | ReadCmd → Debugger.Read |
| w | 写入（弹窗输入）| PopupInput → Debugger.Write |
| s | 扫描当前协议 | StartScanCmd（spinner）|
| / | 搜索过滤 | ToggleFilter |
| ? | 帮助面板 | ToggleHelp |
| q / Esc | 退出 / 返回 | Quit / Back |

## 7. 列宽预算（60% 栏宽，120 列终端 ≈ 72 列）

### 模式 A：默认（BIN 折叠）
`ADDR 6 + HEX 6 + FIELD 16 + 描述 44`（≈22 个全角中文，不溢出）

### 模式 B：调试（x/b 展开 BIN）
`ADDR 6 + HEX 6 + BIN 11 + FIELD 16 + 描述 33`（超出截断 `...`）

## 8. 响应式

| 宽度 | 布局 |
|------|------|
| ≥120 | 两栏 25/75 全功能（分组条 + sparkline）|
| 100-119 | 全功能，sparkline 合并为单柱 |
| 80-99 | 隐藏 instrumentBar，分组条折叠为下拉 |
| <80 | 单栏（设备列表收为弹出层）|

## 8.5 视觉升级（v2 设计，炫度层）

> 风格基准：btop（仪表盘动态感）+ lazygit（精致留白）。所有升级不改变第 6 节交互逻辑。

| 元素 | 实现 | 说明 |
|------|------|------|
| 标题 | `lipgloss.Gradient` Mauve(#CBA6F7)→Sky(#89DCEB) | "tracecli" 渐变字 |
| 面板边框 | 每栏独立色 | 设备栏 Mauve #CBA6F7 / 寄存器表 Teal #94E2D5 / 分组条 Surface0 |
| 焦点栏 | 边框 Lavender #B4BEFE + 标题反色背景 | Focus 状态一眼可见 |
| 状态点 | `tea.Tick` 驱动动画帧 ●○◐◑ | 连接中 Sky 色，就绪 Green，错误 Red |
| 位域 | 半块字符可视化 | bit=1 前景 Green `█`，bit=0 Surface1 `░`，位名高亮 |
| 流量图 | sparkline 字符 ▁▂▃▄▅▆▇█ | 读 Blue / 写 Green 双柱，最近 60 秒 |
| 扫描进度 | spinner + `▰▰▰▱▱ 45%` | 扫描中显示（Sky）|
| Tab 激活 | 圆角胶囊底色 Surface0 #313244 | 激活协议 Tab 胶囊化 |

约束：动画帧统一走 bubbletea 的 Cmd/Tick 机制（不阻塞 Update）；sparkline 宽度自适应；CJK 双宽字符仍按 go-runewidth 计算。

## 9. 实现注记（与项目对接）

### 9.1 已知渲染陷阱（preview 已踩过，必须遵守）

1. **行内元素禁止 Border**：Tab/胶囊只用 Background + Padding 模拟；Border 仅允许用于整面板。嵌套 Border 在 Join 时必然错位重叠。
2. **状态单一来源**：Header、Tab 高亮、状态栏速率等全部从 Model 当前字段渲染；协议切换必须同步更新所有显示点，禁止各自缓存状态。
3. **面板背景必须填满**：每个面板显式 `Width(panelW)` / `Height(panelH)`，内容不足用空格补齐（`lipgloss.PlaceVertical`）。
4. **help 文案由绑定表生成**：快捷键说明只能从 keybindings 表生成，禁止手写第二处，杜绝不一致。

### 9.2 与项目对接

- 协议 Tab 数据源：`protocol.Supported()`
- 设备列表数据源：当前协议 Debugger 的 `Probe(ctx)` 结果（Target 列表）
- 寄存器组：从选中 Target 的 `DevMap.Registers` 按 `addr>>5` 切片
- 位域数据：`DevMap.Register.Bits`
- 状态栏连接态：订阅 `app` 事件总线（EventStateChanged）
- 扫描流程：s → app 派发 scan 命令 → spinner 开始 → Probe 完成 → 更新列表 + toast
