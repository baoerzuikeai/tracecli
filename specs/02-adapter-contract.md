# 适配器契约（specs/02）

物理通道层。每个适配器实现一个文件，通过注册机制挂载。

## Adapter 接口

```go
package adapter

type Config struct {
    PortName string        // Windows: "COM3"，Linux: "/dev/ttyUSB0"
    BaudRate int           // VCP 模式下波特率通常无意义，保留统一字段
    Latency  time.Duration // USB 延迟计时器设置（CH341 需调小）
}

type Adapter interface {
    Open(cfg Config) error
    Close() error
    Write(b []byte) error
    Read(n int, timeout time.Duration) ([]byte, error) // 定长阻塞读
    Reset() error
    ID() (vendor, serial string)   // 自动识别用
    Latency() time.Duration        // 单命令往返估算，UI 显示与超时计算用
}
```

## 实现清单（按优先级）

| 适配器 | 文件 | 说明 |
|--------|------|------|
| mock | `mock.go` | 必须最先实现，所有单测依赖它 |
| ch341_vcp | `ch341_vcp.go` | 命令包格式见 CH341 datasheet 附录 A（CMD_ 系列） |
| ftdi_vcp | `ftdi_vcp.go` | go-serial 透传，波特率无意义 |

## 实现要求

- 所有实现必须配 mock 串口单测（`*_test.go`），验证：命令包字节正确、定长读取、超时、Reset 行为
- `Read` 必须用 `io.ReadFull` 语义实现，禁止依赖单次 `Read` 返回值
- 串口库统一走 `go.bug.st/serial`（Windows 兼容性最好）
- Windows 上串口枚举：WMI 查 `Win32_SerialPort` 获取设备描述（区分 CH341/FTDI/蓝牙虚拟串口）
- 掉线（IO 错误）通过返回错误传播给 app 层重连状态机，禁止内部吞掉或自动重开

## 验收标准

1. mock 单测覆盖 Open/Close/Read 超时/Reset
2. ch341_vcp 的 I2C 命令包字节级单测（比对 datasheet 示例）
3. 所有测试通过 `go test ./... -race`
