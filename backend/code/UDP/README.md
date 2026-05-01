# UDP Socket 练习

## 运行

打开两个终端：

```bash
cd /home/xyc20060125/projects/go_code/backend/code/UDP
go run nodeA/A.go
```

```bash
cd /home/xyc20060125/projects/go_code/backend/code/UDP
go run nodeB/B.go
```

`nodeA/A.go` 监听 `127.0.0.1:9001`，并向
`127.0.0.1:9002` 发送 UDP 数据包。

`nodeB/B.go` 监听 `127.0.0.1:9002`，并向
`127.0.0.1:9001` 发送 UDP 数据包。

两个程序都会每 2 秒发送一次 `Hello`，并打印收到的所有 UDP 数据包。

目录结构：

```text
UDP/
├── README.md
├── nodeA/
│   └── A.go
└── nodeB/
    └── B.go
```

## API 说明

`net.ListenUDP("udp", localAddr)` 会把一个 UDP socket 绑定到本地 IP 和端口。
调用成功后，当前进程就可以接收发送到该地址的数据报。

`conn.ReadFromUDP(buf)` 会阻塞等待，直到收到一个 UDP 数据报。它会返回读取到的
字节数以及发送方的 UDP 地址。

`conn.WriteToUDP(data, peerAddr)` 会向目标 UDP 地址发送一个 UDP 数据报。UDP 是
无连接协议，因此这个调用不会创建持久连接。

## STUN 如何发现公网 IP 和端口

STUN 通常用于让 NAT 后面的设备发现 NAT 为自己分配的公网 UDP 地址。

基本流程：

1. 客户端向公网 STUN 服务器发送一个 UDP STUN Binding Request。
2. 家用或办公网络中的 NAT 创建一条映射，例如
   `192.168.1.10:53000 -> 203.0.113.8:42001`。
3. STUN 服务器看到的数据包来源地址是 `203.0.113.8:42001`。
4. 服务器返回 Binding Response，其中包含它观察到的公网地址；该地址通常放在
   `XOR-MAPPED-ADDRESS` 属性中。
5. 客户端解析响应，从而得到自己的公网 UDP IP 和端口。

重要限制：如果 NAT 是对称 NAT，那么每个目标地址对应的公网映射都可能不同。
这种情况下，从某个 STUN 服务器发现的地址，可能无法直接复用于另一个对端。

下面是一个使用 STUN 库的最小 Go 示例：

```go
package main

import (
	"fmt"

	"github.com/pion/stun"
)

func main() {
	client, err := stun.Dial("udp", "stun.l.google.com:19302")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	request := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var publicAddr stun.XORMappedAddress
	var resultErr error

	err = client.Do(request, func(event stun.Event) {
		if event.Error != nil {
			resultErr = event.Error
			return
		}

		resultErr = publicAddr.GetFrom(event.Message)
	})
	if err != nil {
		panic(err)
	}
	if resultErr != nil {
		panic(resultErr)
	}

	fmt.Printf("public UDP address: %s:%d\n", publicAddr.IP, publicAddr.Port)
}
```

在单独的模块中运行：

```bash
go get github.com/pion/stun
go run main.go
```
