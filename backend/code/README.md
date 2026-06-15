# NetWeaver Backend

这份说明只描述当前 `backend/code` 目录下真实存在、并且已经和仓库现状对齐的后端启动方式。

## 目录定位

- Go 模块根目录是 `backend/code`
- Controller 启动入口是 `backend/code/cmd/controller`
- Node 启动入口是 `backend/code/cmd/node`

也就是说，后端相关命令推荐先进入：

```bash
cd backend/code
```

后面的 `go run ./cmd/...`、`go test ./...` 都以这个目录为基准。

## 启动前先理解的 4 个点

1. Controller 既提供 HTTP API，也会同时启动 UDP Relay。
2. Dashboard 登录、Node 注册和 P2P / Relay 建链都依赖当前进程里的配置值。
3. 如果你还在使用默认账号、默认 JWT、默认 PSK、默认 Relay 公网地址，程序启动时会主动打印警告。
4. 真实多机联调时，最容易填错的是 `NETWEAVER_RELAY_ADDR`。如果不填公网 / 局域网可达地址，节点可能会把 Relay 当成自己本机。

## 推荐环境变量

真实部署或多机联调前，建议至少设置这些环境变量：

```bash
export NETWEAVER_DASHBOARD_USERNAME=admin
export NETWEAVER_DASHBOARD_PASSWORD='请换成你们自己的密码'
export NETWEAVER_JWT_SECRET='请换成你们自己的 JWT 密钥'
export NETWEAVER_NODE_PSK='请换成你们自己的节点 PSK'
export NETWEAVER_RELAY_ADDR='控制器所在机器的真实可达 IP'
export NETWEAVER_RELAY_PORT=9000
export NETWEAVER_STUN_SERVERS='stun.l.google.com:19302,stun1.l.google.com:19302,stun2.l.google.com:19302'
```

说明：

- `NETWEAVER_RELAY_ADDR` 是节点收到的 Relay 公网地址，不是监听地址写法。
- `NETWEAVER_RELAY_PORT` 对应 Controller 里 UDP Relay 的端口。
- `NETWEAVER_STUN_SERVERS` 可以继续用默认值，也可以改成你们自己的 STUN 方案。

## 启动 Controller

进入后端目录：

```bash
cd backend/code
```

启动 Controller：

```bash
go run ./cmd/controller
```

默认行为：

- HTTP 监听 `:8080`
- UDP Relay 监听 `:9000`
- Dashboard 接口和 Node 接口都由这个进程提供

如果要改监听端口：

```bash
go run ./cmd/controller -addr :9090 -relay-addr :9100
```

## 验证 Controller 是否启动成功

测试健康检查：

```bash
curl http://127.0.0.1:8080/ping
```

预期返回：

```json
{"msg":"pong"}
```

测试登录接口是否存在：

```bash
curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"change-me"}'
```

如果你已经修改了环境变量，这里就把账号密码换成你自己的值。

## Dashboard 当前使用的接口

前端控制台当前会用到这些接口：

- `POST /api/v1/auth/login`
- `GET /api/v1/dashboard/stats`
- `GET /api/v1/dashboard/nodes`
- `GET /api/v1/dashboard/edges`
- `GET /api/v1/dashboard/nodes/{node_id}/metrics?target_id={target_id}&time_range=1h|12h|24h`

注意：

- 登录接口不需要 JWT。
- `dashboard/*` 接口需要 `Authorization: Bearer <jwt>`。
- `nodes/*` 节点接口需要 `Authorization: Bearer <psk>`。

## 启动 Node 运行模式

Node 真正接 Controller 联调时，请使用 `run` 子命令。

基本写法：

```bash
cd backend/code
go run ./cmd/node run -controller http://127.0.0.1:8080
```

更完整的常用写法：

```bash
go run ./cmd/node run \
  -controller http://127.0.0.1:8080 \
  -hostname node-a \
  -local-ip 192.168.1.10 \
  -data-addr 0.0.0.0:0 \
  -tun-name tuno \
  -tun-auto-config \
  -tun-auto-cleanup
```

常见参数说明：

- `-controller`：Controller 的 HTTP 地址
- `-machine-id`：稳定机器标识；不填时会自动推导
- `-hostname`：节点显示名称
- `-local-ip`：节点本机内网地址
- `-nat-type`：手动覆盖 NAT 类型；默认 `Unknown`
- `-public-ip` / `-public-port`：手动上报公网地址；通常可不填
- `-stun-servers`：覆盖 STUN 列表；默认会读取 `NETWEAVER_STUN_SERVERS`
- `-psk`：覆盖节点 PSK；默认会读取 `NETWEAVER_NODE_PSK`
- `-tun`：是否启用 TUN 数据面，默认 `true`
- `-tun-name`：TUN 网卡名，默认 `tuno`
- `-tun-auto-config`：节点注册拿到 `virtual_ip` 后，自动执行 `ip addr replace ...` 和 `ip link set ... up`
- `-tun-auto-cleanup`：节点退出时，自动删除刚才自动加上的 `virtual_ip`，并把网卡置为 down
- `-tun-prefix`：自动配置 TUN 时使用的前缀长度，默认从 `10.0.0.0/16` 推导出 `16`

## TUN 模式说明

当前项目里，Node `run` 模式默认会尝试打开 TUN 数据面。

这意味着：

- 如果当前机器没有 TUN 权限，节点仍然可能注册成功并显示在线。
- 但这不代表数据面一定可用。
- 程序会在日志里打印 `TUN data plane disabled: ...` 或 `TUN data plane enabled: ...`。

如果你只是先验证控制面联调，可以临时关闭 TUN：

```bash
go run ./cmd/node run -controller http://127.0.0.1:8080 -tun=false
```

如果你要减少手工命令，推荐直接这样运行：

```bash
go run ./cmd/node run \
  -controller http://127.0.0.1:8080 \
  -hostname node-a \
  -local-ip 192.168.1.10 \
  -data-addr 0.0.0.0:9101 \
  -tun-name nw0 \
  -tun-auto-config \
  -tun-auto-cleanup
```

这套参数的效果是：

- Node 注册成功拿到 `virtual_ip` 后，会自动给 `nw0` 配置该地址并把网卡拉起。
- Node 退出时，会自动把这次自动配置的地址删掉，并把 `nw0` 置为 down。
- 联调时不再需要手动执行 `ip addr replace ...` 和 `ip link set ... up/down`。

如果你要真正验证 TUN 数据面，请确保当前 Linux / WSL / 容器环境允许创建 TUN，并按日志提示配置虚拟网卡地址。

### WSL / P2P 说明

当前代码已经可以把 TUN 网卡的“配置 / 解除配置”自动化到 `node run` 参数里，但这只能降低操作门槛，不能单靠代码保证 WSL 一定打成 P2P。

如果你在 WSL 里联调，建议：

1. 尽量使用 WSL mirrored networking，而不是默认 NAT。
2. 给 `-data-addr` 指定固定端口，例如 `0.0.0.0:9101`。
3. 确认 Windows 防火墙放行该 UDP 端口。
4. 如果仍然打洞失败，不代表系统不可用，当前链路会回退到 Relay。

## P2P 实验模式

`p2p` 模式是独立的 UDP 双节点实验，不接 Controller。

终端 1：

```bash
cd backend/code
go run ./cmd/node p2p -profile A
```

终端 2：

```bash
cd backend/code
go run ./cmd/node p2p -profile B
```

## 编译检查

在 `backend/code` 目录下执行：

```bash
go test ./...
```

当前仓库测试文件不多，但这个命令至少可以验证所有 Go 包是否还能正常编译。

## 服务器一键部署

如果你们已经准备把 Controller 和前端控制台一起放到 Linux 服务器上，当前仓库已经提供一套最小可用的一键部署脚本。

推荐流程：

```bash
cd NetWeaver
bash scripts/deploy_server.sh
```

这个脚本会做这些事：

- 检查服务器上是否有 `go` 和 `npm`
- 在 `backend/code` 下执行 `go test ./...`
- 编译 `controller` 可执行文件
- 在 `frontend` 下执行 `npm ci` 和生产构建
- 生成服务器部署目录 `deploy/server/`
- 自动生成环境变量模板 `deploy/server/netweaver-server.env`
- 自动生成启动脚本 `deploy/server/run/start-controller.sh`
- 自动生成可选的 `systemd` unit 文件 `deploy/server/netweaver-controller.service`

### 部署产物目录

脚本跑完后，主要关注这些目录：

- `deploy/server/controller/bin/netweaver-controller`
- `deploy/server/console/web/`
- `deploy/server/netweaver-server.env`
- `deploy/server/run/start-controller.sh`

### controller 托管前端控制台

当前 `controller` 已支持通过参数托管前端静态页面：

```bash
./netweaver-controller \
  -addr :8080 \
  -relay-addr :9000 \
  -console-dir /path/to/console/web \
  -console-base /console
```

这意味着：

- API 仍然走 `/api/v1/...`
- Relay 仍然由同一个进程提供
- 前端控制台页面会挂在 `http://<server-ip>:8080/console/`
- 不需要再额外启动一个 `vite preview` 或 Python 静态文件服务器

### 环境变量文件

首次执行 `scripts/deploy_server.sh` 时，如果 `deploy/server/netweaver-server.env` 不存在，脚本会自动生成模板。

这套部署方式的分工是：

- `scripts/deploy_server.sh` 负责读取环境变量、执行编译、生成部署目录和启动脚本。
- `deploy/server/netweaver-server.env` 负责保存你们自己的服务器配置。

也就是说，脚本和 `.env` 文件是配合使用的，不是二选一。

严格来说，当前后端大多数配置都有默认值，所以“代码层面”并不是不填就一定启动失败。
但从真实联调和落地角度看，下面这些项已经可以分成“必须改”和“可按需改”。

#### 多机部署前必须修改

- `NETWEAVER_DASHBOARD_PASSWORD`
- `NETWEAVER_JWT_SECRET`
- `NETWEAVER_NODE_PSK`
- `NETWEAVER_RELAY_ADDR`

原因：

- 前三项如果不改，系统虽然能跑，但仍然是在用开发默认密码和默认密钥，不适合真实部署。
- `NETWEAVER_RELAY_ADDR` 如果不改成服务器真实可达 IP，其他机器上的节点可能会错误地把 Relay 当成它们自己的本机地址。

#### 建议根据实际环境确认

- `NETWEAVER_STUN_SERVERS`
- `NETWEAVER_RELAY_PORT`
- `NETWEAVER_RELAY_LISTEN`
- `NETWEAVER_HTTP_ADDR`

原因：

- `NETWEAVER_STUN_SERVERS` 默认公共列表通常可用，但如果校园网、机房或目标环境有限制，就要换成你们自己的方案。
- `NETWEAVER_RELAY_PORT` 是节点收到的 Relay 对外端口，通常要和 `NETWEAVER_RELAY_LISTEN` 的监听端口保持一致。
- `NETWEAVER_HTTP_ADDR` 和 `NETWEAVER_RELAY_LISTEN` 是否要改，取决于你们服务器的端口规划。

#### 可保留默认值，按需再改

- `NETWEAVER_DASHBOARD_USERNAME`
- `NETWEAVER_CONSOLE_BASE`

原因：

- `NETWEAVER_DASHBOARD_USERNAME` 默认 `admin` 可以正常使用，只是从安全性角度看，后续也可以改。
- `NETWEAVER_CONSOLE_BASE` 默认 `/console`，对应浏览器访问 `http://<server-ip>:8080/console/`，大多数情况下保持默认最省事。

#### 前端构建相关配置

如果你们让 controller 托管前端控制台，通常还会保持：

- `VITE_API_BASE_URL=` 留空
- `VITE_APP_BASE=/console/`

这样前端会继续请求同域名下的 `/api/...`，最适合服务器同域部署。

这里再强调一下：

- `VITE_API_BASE_URL` 在“后端和前端由同一个 controller 托管”的方案里，通常留空就是正确配置。
- `VITE_APP_BASE` 要和 `NETWEAVER_CONSOLE_BASE` 对应上。
  例如后端挂载路径是 `/console`，前端构建基础路径就写 `/console/`。

### 手动启动

部署脚本跑完后，可以先不装 `systemd`，直接手动启动：

```bash
bash deploy/server/run/start-controller.sh
```

### 安装 systemd 服务

如果你们希望服务器开机自启或长期常驻运行，再执行：

```bash
sudo bash scripts/install_systemd.sh
sudo systemctl start netweaver-controller.service
sudo systemctl status netweaver-controller.service
```

### 访问结果

部署成功后，至少应能访问：

- `http://<server-ip>:8080/ping`
- `http://<server-ip>:8080/console/`

## 多机联调最容易填错的地方

1. 前端请求的后端地址，要填 Controller 的 HTTP 地址，例如 `http://192.168.1.23:8080`。
2. Node `-controller` 参数，也要填同一个 Controller HTTP 地址。
3. `NETWEAVER_RELAY_ADDR` 要填 Controller 所在机器对其他节点可达的真实 IP，不能继续写 `127.0.0.1`。
4. 节点侧 `-local-ip` 应该尽量填写当前机器真实参与联调的网卡 IP，避免上报错误地址。
5. 如果图上看到节点在线但没有链路，优先检查 STUN、Relay 地址、PSK 和 TUN 日志，而不是先怀疑前端页面。
