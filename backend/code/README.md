# Netweaver Backend

这是一个 Go 后端项目，主要包含两个可运行入口：

- HTTP 控制器服务：`cmd/controller`
- 节点程序：`cmd/node`

项目运行目录是：

```bash
cd /home/xyc20060125/projects/go_code/backend/code
```

> 注意：当前机器上的 `go` 命令没有加入 `PATH`，但 Go 已安装在 `/usr/local/go/bin/go`。下面的命令默认使用完整路径。

## 运行 HTTP 后端服务

启动控制器服务：

```bash
cd /home/xyc20060125/projects/go_code/backend/code
/usr/local/go/bin/go run ./cmd/controller
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
curl http://127.0.0.1:8080/nodes
```

注册一个节点：

```bash
curl -X POST http://127.0.0.1:8080/nodes \
  -H "Content-Type: application/json" \
  -d '{"id":"node1","address":"127.0.0.1:9001"}'
```

如果需要更换端口，例如监听 `9090`：

```bash
/usr/local/go/bin/go run ./cmd/controller -addr :9090
```

## 运行 P2P 节点

`cmd/node` 支持 `p2p` 模式。需要打开两个终端分别启动 NodeA 和 NodeB。

终端 1：

```bash
cd /home/xyc20060125/projects/go_code/backend/code
/usr/local/go/bin/go run ./cmd/node p2p -profile A
```

终端 2：

```bash
cd /home/xyc20060125/projects/go_code/backend/code
/usr/local/go/bin/go run ./cmd/node p2p -profile B
```

默认配置：

- NodeA 监听 `127.0.0.1:9001`
- NodeB 监听 `127.0.0.1:9002`
- 两边每隔 2 秒互相发送 `"Hello"`

也可以手动指定节点参数：

```bash
/usr/local/go/bin/go run ./cmd/node p2p \
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
cd /home/xyc20060125/projects/go_code/backend/code
sudo /usr/local/go/bin/go run ./cmd/node -bootstrap -ifname tuno
```

然后配置网卡：

```bash
sudo ip addr add 10.23.0.1/24 dev tuno
sudo ip link set tuno up
```

之后普通运行：

```bash
/usr/local/go/bin/go run ./cmd/node -ifname tuno
```

也可以直接以 root 权限运行：

```bash
sudo /usr/local/go/bin/go run ./cmd/node
```

## 配置 Go PATH

为了以后不用每次写 `/usr/local/go/bin/go`，可以把 Go 加入 `PATH`。

当前终端临时生效：

```bash
export PATH=/usr/local/go/bin:$PATH
```

之后就可以直接使用：

```bash
go run ./cmd/controller
```

如果希望长期生效，可以把下面这行加入 shell 配置文件，例如 `~/.bashrc`：

```bash
export PATH=/usr/local/go/bin:$PATH
```

然后重新加载配置：

```bash
source ~/.bashrc
```

## 编译验证

检查整个项目是否能正常编译：

```bash
cd /home/xyc20060125/projects/go_code/backend/code
/usr/local/go/bin/go test ./...
```

当前项目没有测试文件，但该命令可以确认所有包都能正常编译。
