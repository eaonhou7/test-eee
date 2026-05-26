# 静态部署说明

本方案用于低内存机器，例如 2C2G。运行时只保留：

```text
Go 后端二进制 + web/dist 静态文件 + MySQL 8
```

不再运行 `vite` 开发服务，也不需要前端常驻 Node 进程。

## 1. 适用场景

- 小数据量演示、内网使用、轻量业务。
- 服务器内存较小，希望减少常驻进程。
- 前端代码不需要热更新，改完后重新构建 `web/dist`。

## 2. 准备配置

需要先安装：

```text
Go、Node.js、npm、MySQL 8、curl、lsof、python3
```

如果前端和 Go 二进制都提前在其他机器构建好，目标机运行时只需要 MySQL 8 和编译后的 Go 二进制。

确认 MySQL 8 已启动，并准备后端本地配置：

```bash
cp server/config.local.example.yaml server/config.local.yaml
```

然后修改 `server/config.local.yaml` 里的 MySQL 配置：

```yaml
mysql:
    path: 127.0.0.1
    port: "3306"
    db-name: amazon_admin
    username: root
    password: 你的 MySQL 密码
```

默认后端端口是 `8888`：

```yaml
system:
    addr: 8888
```

## 3. 启动静态部署

macOS / Linux 在项目根目录执行：

```bash
./scripts/static-up.sh
```

Windows PowerShell 在项目根目录执行：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\scripts\static-up.ps1
```

脚本会自动执行：

```text
1. npm install
2. npm run build:static
3. go build
4. 启动 Go 后端
5. 由 Go 后端托管 web/dist 静态文件
```

启动后访问：

```text
http://127.0.0.1:8888/#/login
```

健康检查：

```bash
curl http://127.0.0.1:8888/health
```

正常返回：

```json
"ok"
```

## 4. 停止服务

macOS / Linux：

```bash
./scripts/static-down.sh
```

Windows PowerShell：

```powershell
.\scripts\static-down.ps1
```

## 5. 2G 内存机器建议

2G 机器上不建议直接构建前端，因为 `npm run build` 峰值内存可能比较高。更稳的方式是在开发机或 CI 上构建：

```bash
cd web
npm install --prefer-offline --no-audit --no-fund
npm run build:static
```

把生成的 `web/dist` 放到目标服务器后，在目标服务器执行：

```bash
STATIC_BUILD=0 ./scripts/static-up.sh
```

Windows PowerShell：

```powershell
$env:STATIC_BUILD = "0"
.\scripts\static-up.ps1
```

这样目标服务器只构建 Go 后端并运行服务，Node 不作为常驻进程。

如果 Go 二进制也已经提前构建好，可以跳过后端构建：

```bash
STATIC_BUILD=0 SERVER_BUILD=0 ./scripts/static-up.sh
```

Windows PowerShell：

```powershell
$env:STATIC_BUILD = "0"
$env:SERVER_BUILD = "0"
.\scripts\static-up.ps1
```

## 6. 环境变量

| 变量 | 默认值 | 说明 |
|---|---:|---|
| `STATIC_BUILD` | `1` | 是否执行 `npm run build:static`，设为 `0` 时复用现有 `web/dist` |
| `SERVER_BUILD` | `1` | 是否检查并构建 Go 二进制 |
| `API_PORT` | `8888` | 脚本健康检查端口，需要和 `server/config.local.yaml` 的 `system.addr` 一致 |
| `SERVER_READY_TIMEOUT` | `120` | 等待服务启动的秒数 |

脚本会设置：

```bash
GVA_STATIC_ROOT=web/dist
```

后端检测到该变量后，会把 `web/dist` 作为静态站点托管。

## 7. 与开发模式的区别

开发模式：

```text
Go 后端 8888 + Vite 前端 8080 + MySQL
```

静态部署：

```text
Go 后端 8888 同时提供 API 和 web/dist + MySQL
```

所以静态部署只访问一个端口：

```text
http://127.0.0.1:8888/#/login
```
