# 协议契约（specs/03）

协议层核心。新增协议只加一个文件 + 注册，UI 零改动。

## 通用数据结构

```go
package protocol

type Target struct {
    Protocol string // "i2c" / "spi" / "uart" ...
    Address  uint16 // 协议相关寻址（I2C 7位地址、SPI CS 编号）
    Name     string // 识别结果，未知为 "Unknown@0xXX"
    DevMap   *device.DevMap // 指纹命中后挂载的寄存器定义，可为 nil
}

type Capability uint32

const (
    CapReadWrite   Capability = 1 << iota // 读写寄存器
    CapScan                                // 支持地址扫描
    CapFingerprint                         // 支持指纹识别
    CapReset                               // 支持设备重置
    CapWaveform                            // 支持波形/采样（SPI 抓时序用）
    CapBatch                               // 支持批量/脚本
)
```

## Debugger 接口

```go
type Debugger interface {
    Name() string
    Probe(ctx context.Context) ([]Target, error)
    Read(t Target, addr uint16, n int) ([]byte, error)
    Write(t Target, addr uint16, data []byte) error
    Reset(t Target) error
    Capabilities() []Capability
}
```

## 注册表（扩展入口）

```go
package protocol

type Factory func(a adapter.Adapter) (Debugger, error)

var registry = map[string]Factory{}

func Register(name string, f Factory) { registry[name] = f }
func Supported() []string             // UI 据此动态生成 Tab
func New(name string, a adapter.Adapter) (Debugger, error)

// 新增协议：新建 protocol/<name>.go，文件内 init() 调 Register，
// 禁止修改 registry 本身、UI 层、Target 结构。
```

## 协议特有参数

协议级参数（如 I2C 时钟频率、SPI 的 CPOL/CPHA、UART 波特率）放各协议自有的 `Options` 结构，**由 config.yaml 按协议节点配置**，不进通用接口。

```yaml
# config.yaml 中按协议分节
protocols:
  i2c:
    clock: 100000
  spi:
    mode: 0
    clock: 1000000
```

## 实现要求

- 每个协议必须实现 `Probe`（扫描）与 `Read/Write`，并配 mock adapter 单测
- 禁止协议特有概念泄漏进 Target / Capabilities
- Capabilities 只加位不改位

## 验收标准

1. `protocol/i2c.go` 用 mock adapter 跑通：读/写/扫描/指纹匹配全链路
2. `protocol/uart.go` 验证透传（v0.3）
3. 新协议加入后 `go build ./...` 通过，UI 无需改动
