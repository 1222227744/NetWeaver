# Netweaver Backend

这是一个 Go 后端项目，主要包含两个可运行入口：

- HTTP 控制器服务：`cmd/controller`
- 节点程序：`cmd/node`

项目运行目录是：

```bash
cd /home/xyc20060125/projects/temp/NetWeaver
```

> 当前仓库根目录已经配置了 `go.work`，可以直接在仓库根目录运行后端 Go 模块。

## 运行 HTTP 后端服务

启动控制器服务：

```bash
cd /home/xyc20060125/projects/temp/NetWeaver
go run ./backend/code/cmd/controller
```

默认监听地址是：

```text
http://127.0.0.1:8080
```

测试服务是否正常：

```bash
curl http://127.0.0.1:8080/ping
```

正常会返回类似：

```json
{"msg":"pong"}
```

查看节点列表：

```bash
curl http://127.0.0.1:8080/api/v1/dashboard/nodes
```

查看全局统计：

```bash
curl http://127.0.0.1:8080/api/v1/dashboard/stats
```

注册一个节点：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/nodes/register \
  -H "Content-Type: application/json" \
  -d '{"machine_id":"e4:5f:01:aa:bb:cc","hostname":"ubuntu-server-01","os":"linux","local_ip":"192.168.1.10"}'
```

节点心跳：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/nodes/{node_id}/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"nat_type":"Full Cone","public_ip":"203.0.113.5","public_port":54321,"current_rx_bytes":1024560,"current_tx_bytes":2048000}'
```

拉取对等节点：

```bash
curl http://127.0.0.1:8080/api/v1/nodes/{node_id}/peers
```

如果需要更换端口，例如监听 `9090`：

```bash
go run ./backend/code/cmd/controller -addr :9090
```

## 运行 P2P 节点

`cmd/node` 支持 `p2p` 模式。需要打开两个终端分别启动 NodeA 和 NodeB。

终端 1：

```bash
cd /home/xyc20060125/projects/temp/NetWeaver
go run ./backend/code/cmd/node p2p -profile A
```

终端 2：

```bash
cd /home/xyc20060125/projects/temp/NetWeaver
go run ./backend/code/cmd/node p2p -profile B
```

默认配置：

- NodeA 监听 `127.0.0.1:9001`
- NodeB 监听 `127.0.0.1:9002`
- 两边每隔 2 秒互相发送 `"Hello"`

也可以手动指定节点参数：

```bash
go run ./backend/code/cmd/node p2p \
  -name MyNode \
  -local 127.0.0.1:9010 \
  -peer 127.0.0.1:9011 \
  -message "hi" \
  -interval 1s
```

## 运行 TUN 模式

如果只是运行普通 HTTP 后端，可以先不用管 TUN 模式。

TUN 模式会创建虚拟网卡，通常需要 `sudo` 或 `NET_ADMIN` 权限。

一次性初始化：

```bash
cd /home/xyc20060125/projects/temp/NetWeaver
sudo /usr/local/go/bin/go run ./backend/code/cmd/node -bootstrap -ifname tuno
```

然后配置网卡：

```bash
sudo ip addr add 10.23.0.1/24 dev tuno
sudo ip link set tuno up
```

之后普通运行：

```bash
go run ./backend/code/cmd/node -ifname tuno
```

也可以直接以 root 权限运行：

```bash
sudo /usr/local/go/bin/go run ./backend/code/cmd/node
```

## 配置 Go PATH

Go 1.24.0 已安装在 `/usr/local/go`，并且本机 shell 配置已经把 `/usr/local/go/bin` 加入 `PATH`。

如果旧终端里还找不到 `go`，重新打开终端，或者手动加载配置：

```bash
source ~/.profile
```

确认版本：

```bash
go version
```

## 编译验证

检查整个项目是否能正常编译：

```bash
cd /home/xyc20060125/projects/temp/NetWeaver
go test ./backend/code/...
```

当前项目没有测试文件，但该命令可以确认所有包都能正常编译。
