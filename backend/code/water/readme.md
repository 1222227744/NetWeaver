Go TUN Sniffer (基于 water 库)
本目录用于演示：
1. `github.com/songgao/water` 的引入
2. 创建名为 `tuno` 的 TUN 设备
3. 持续打印“发往该网卡网段”的报文 Hex 输出
4. 默认自动应答 ICMP Echo（便于本地验证不丢包）

## 运行环境
- Linux / WSL2
- 有 `/dev/net/tun`
- 有 `ip` 命令（`iproute2`）
- Go 1.24+

## 唯一推荐运行方法（最稳妥）
只使用这一套流程，避免 `sudo go run` 下载依赖超时。

```bash
cd /home/xyc20060125/projects/go_code/backend/code/test

# 1) 普通用户先下载依赖
/usr/local/go/bin/go env -w GOPROXY=https://goproxy.cn,direct
/usr/local/go/bin/go env -w GOSUMDB=sum.golang.google.cn
/usr/local/go/bin/go mod download

# 2) 编译
/usr/local/go/bin/go build -o tun-sniffer .

# 3) 一次性 bootstrap（root）
sudo ./tun-sniffer -bootstrap -ifname tuno

# 4) 配置并启用 tuno
sudo ip addr replace 10.23.0.1/24 dev tuno
sudo ip link set tuno up

# 5) 普通用户启动抓包（此终端会阻塞等待报文）
./tun-sniffer -ifname tuno
```

另开一个终端发包：

```bash
ping -I tuno -c 4 10.23.0.2  # 发四个包
ping -I tuno 10.23.0.2  #  一直发
```

如果你只想抓包、不希望程序自动回包：

```bash
./tun-sniffer -ifname tuno -icmp-echo-reply=false
```

## 验收检查
```bash
ip link show tuno
ip addr show tuno
```

抓包终端应持续打印类似：
```text
[15:04:05.000] packet 84 bytes
00000000  45 00 00 54 ...
```
程序会根据 `tuno` 的 IPv4 地址自动识别网段（例如 `10.23.0.1/24 -> 10.23.0.0/24`），
仅打印目的地址落在该网段内的 IPv4 报文。
默认开启 ICMP 自动应答，所以 `ping -I tuno -c 4 10.23.0.2` 不会再是 100% 丢包。

## 清理（可选）
```bash
sudo ip link delete tuno
```
