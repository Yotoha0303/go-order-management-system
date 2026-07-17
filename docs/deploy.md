# 一键部署说明（stage-10）

本文说明仓库根目录 [`deploy.sh`](../deploy.sh) 的用途、前置条件和部署结果。脚本面向 **Ubuntu/Debian 云主机**，把当前版本的后端完整栈和前端管理台部署到一台服务器上。

## 相对上一版的变化

| 项 | 旧版脚本 | 当前脚本 |
| --- | --- | --- |
| 依赖服务 | MySQL + Redis 为主 | MySQL + Redis + **RabbitMQ**（订单超时取消） |
| 数据库迁移 | 手动 `docker compose run migrate` | Compose 内 `migrate` 成功后才启动 `app` |
| 前端 | `npm run dev` 常驻 Vite | **生产构建** `fronted/dist`，由 Nginx 静态托管 |
| 反代路径 | `/api` + 前端 | `/api`、`/ping`、`/live`、`/readyz`、**`/metrics`** + SPA |
| 配置 | 仅基础密钥 | 兼容慢 SQL、库存对账、订单超时等 stage-10 环境变量 |
| 密钥 | 提示手改 `.env` | 缺失/占位时自动生成 `MYSQL_PASSWORD` 与 `JWT_SECRET` |

## 前置条件

- 全新或可接受改动的 Ubuntu/Debian 服务器
- 能使用 `root` 或 `sudo`
- 出站可访问 Docker Hub / 镜像加速与 npm（装前端构建工具）
- 安全组放行 **22**、**80**（脚本也会配置本机 UFW）

建议在独立云主机上使用；不要在已有重要 Nginx/Docker 业务的机器上直接跑全量脚本。

## 快速使用

```bash
# 1. 克隆仓库
git clone https://github.com/Yotoha0303/go-order-management-system.git
cd go-order-management-system

# 2. 执行一键部署（需要 root/sudo）
chmod +x deploy.sh
sudo ./deploy.sh
```

部署过程中若未设置 `NONINTERACTIVE=1`，脚本会在生成/补齐 `.env` 后暂停一次，方便你检查密码与密钥。

### 非交互模式

```bash
sudo NONINTERACTIVE=1 ./deploy.sh
```

### 常用可选环境变量

| 变量 | 默认 | 含义 |
| --- | --- | --- |
| `NONINTERACTIVE` | `0` | `1` 时不暂停等待编辑 `.env` |
| `SKIP_SYSTEM_UPDATE` | `0` | `1` 时跳过 `apt-get upgrade` |
| `SKIP_DOCKER_MIRROR` | `0` | `1` 时不写入国内 Docker 镜像加速 |
| `SKIP_FIREWALL` | `0` | `1` 时不改 UFW |
| `SKIP_FRONTEND` | `0` | `1` 时不构建前端（仅后端 + API 反代） |
| `API_PORT` | `8082` | 本机后端监听端口（与 Compose 一致） |

示例：跳过系统升级与镜像加速，直接部署：

```bash
sudo NONINTERACTIVE=1 SKIP_SYSTEM_UPDATE=1 SKIP_DOCKER_MIRROR=1 ./deploy.sh
```

## 脚本做了什么

1. 安装基础包：`curl`、`git`、`ufw`、`nginx` 等  
2. 安装并启动 Docker Engine（若尚未安装）  
3. 可选配置 Docker 国内 registry mirror（仅当 `/etc/docker/daemon.json` 不存在时写入）  
4. UFW 放行 22/80  
5. 从 `.env.example` 生成 `.env`，补齐 stage-10 配置，必要时生成数据库密码和 JWT  
6. `docker compose --env-file .env up -d --build --wait`  
   - 启动 MySQL、Redis、RabbitMQ  
   - 执行 Goose 迁移（含 `operation_logs` 等）  
   - 启动 API、订单超时 worker、库存对账 worker（若配置开启）  
7. 安装 Node.js 20（若需要），构建 `fronted` 生产静态资源  
8. 配置 Nginx：静态前端 + 同源反代 API/健康检查/metrics  
9. 本机探测 `/ping`、`/live`、`/readyz`、`/metrics`

## 部署后访问

假设公网 IP 为 `x.x.x.x`：

| 地址 | 说明 |
| --- | --- |
| `http://x.x.x.x/` | 前端管理台 |
| `http://x.x.x.x/api/v1/...` | 业务 API |
| `http://x.x.x.x/readyz` | 就绪检查 |
| `http://x.x.x.x/metrics` | Prometheus 文本指标 |

本机直连后端（默认不经过域名）：

```bash
curl http://127.0.0.1:8082/readyz
curl http://127.0.0.1:8082/metrics
```

## 运维命令

```bash
# 查看容器
docker compose --env-file .env ps

# 跟随应用日志
docker compose --env-file .env logs -f app

# 停止并移除容器（保留数据卷）
docker compose --env-file .env down

# 重新部署后端镜像
docker compose --env-file .env up -d --build --wait
```

重新构建前端并刷新 Nginx 静态目录：

```bash
cd fronted
npm ci
npm run build
sudo nginx -t && sudo systemctl reload nginx
```

## 安全注意

- `.env` 含数据库密码和 JWT，不要提交到 Git  
- MySQL **默认不映射宿主端口**；Redis/RabbitMQ 虽在 Compose 中映射，但 UFW 默认只放行 22/80，外网不可达  
- 生产环境请自行更换 RabbitMQ 默认口令，并考虑 HTTPS（证书 + 443）  
- 首次部署后请自行注册用户，并按项目 RBAC 设计授予 admin 角色后再使用管理接口

## 与本地开发部署的区别

| 场景 | 推荐方式 |
| --- | --- |
| 本机开发 | `make infra-up` + `make migrate-up` + `make run` + `cd fronted && npm run dev` |
| 本机 Docker 全栈 | `make docker-up` |
| 云主机演示/交付 | `./deploy.sh` |

更完整的本地说明见 [README.md](../README.md)。
