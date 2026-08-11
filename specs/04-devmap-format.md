# 设备库格式（specs/04）

数据驱动扩展：新增设备 = 加一个 yaml，禁止写 Go 代码。

## YAML Schema

```yaml
# devmaps/mpu6050.yaml 示例
device: MPU6050
vendor: TDK InvenSense
addresses: [0x68, 0x69]      # 自动识别候选地址
fingerprint:                  # 识别指纹：读该寄存器应为此值（可多个）
  - reg: 0x75
    value: 0x68
registers:
  - name: WHO_AM_I
    addr: 0x75
    rw: r                      # r / w / rw
    reset: 0x68                # 重置后期望值（CapReset 时用于校验）
  - name: CTRL1
    addr: 0x6B
    rw: rw
    default: 0x00
    bits:                      # 位域定义（TUI 里按位显示）
      - {name: DEVICE_RESET, bit: 7, rw: w}
      - {name: SLEEP, bit: 6, rw: rw}
      - {name: RANGE, bit: [3, 4], rw: rw}   # 连续位域
operations:                    # 复杂设备操作序列（v0.5 脚本引擎消费）
  - {name: FULL_RESET, steps: [{type: write, reg: 0x6B, data: [0x80]}, {type: delay, ms: 100}]}
```

## 字段规则

| 字段 | 必填 | 说明 |
|------|------|------|
| device | 是 | 设备名（显示名 + 匹配名） |
| vendor | 否 | 厂商，TUI 悬停提示 |
| addresses | 是 | 候选地址，Probe 命中任何一个即识别 |
| fingerprint | 否 | 存在则用于识别；不存在则仅靠地址匹配 |
| registers | 是 | 寄存器树（层级浏览的数据源） |
| operations | 否 | 脚本序列，v0.5 才消费 |

## 加载规则

- 目录 `devmaps/*.yaml` 全部加载进索引，文件名即设备键
- 相同设备名后加载覆盖先加载（允许用户放私有库覆盖内置库）
- 解析失败：跳过该文件并记录警告，不影响启动

## 验收标准

1. 内置 mpu6050.yaml 可被 `internal/device` 正确解析
2. 指纹匹配单测：伪扫描序列 → 命中/未命中两种路径
3. 非法 yaml 文件不导致崩溃，仅告警
