# Windows 本地部署指南

本文档用于在 Windows 10/11 上从 0 到 1 启动本项目的本地环境。流程采用 Windows 原生 PowerShell，不依赖 WSL2 或 Docker。

低内存机器优先使用 `scripts\static-up.ps1` 和 `scripts\static-down.ps1`：前端先构建成 `web\dist`，运行时只保留 Go 后端和 MySQL，不常驻 Vite/Node。需要前端热更新开发时，再使用 `scripts\dev-up.ps1` 和 `scripts\dev-down.ps1`。

## 1. 环境准备

### 1.1 安装必需软件

请先安装以下软件，并在安装时勾选加入 `PATH`。

| 软件 | 建议版本 | 用途 |
| --- | --- | --- |
| Git | 2.40+ | 拉取项目代码 |
| Go | 1.24.2 或兼容更新版本 | 编译和运行后端 |
| Node.js | 20 LTS 或更新 LTS | 运行前端 |
| npm | 随 Node.js 安装 | 安装前端依赖 |
| MySQL | 8.0.x | 本地业务数据库 |

安装完成后，重新打开 PowerShell，执行：

```powershell
git --version
go version
node -v
npm -v
mysql --version
```

每条命令都应输出版本号。如果提示“不是内部或外部命令”，说明对应软件没有加入 `PATH`，需要重新配置环境变量或重装时勾选 PATH 选项。

### 1.2 准备工作目录

建议把项目放在纯英文路径，避免中文路径、空格路径导致部分工具解析异常。

```powershell
mkdir C:\workspace
cd C:\workspace
```

## 2. 获取项目代码

```powershell
git clone https://github.com/eaonhou7/test-eee.git
cd test-eee
```

确认项目结构：

```powershell
dir
dir server
dir web
```

应能看到根目录下的 `server`、`web`、`scripts`、`extensions` 等目录。

## 3. 配置 MySQL

### 3.1 启动 MySQL 服务

如果 MySQL 是通过官方安装包安装的，通常会注册为 Windows 服务。以管理员身份打开 PowerShell，执行：

```powershell
Get-Service *mysql*
Start-Service MySQL80
```

如果服务名不是 `MySQL80`，以上一步 `Get-Service *mysql*` 输出的实际服务名为准。

### 3.2 验证 root 账号连接

```powershell
mysqladmin -h127.0.0.1 -P3306 -uroot -p ping
```

输入安装 MySQL 时设置的 root 密码。成功时应输出：

```text
mysqld is alive
```

### 3.3 创建本地数据库

进入 MySQL：

```powershell
mysql -h127.0.0.1 -P3306 -uroot -p
```

执行：

```sql
CREATE DATABASE IF NOT EXISTS amazon_admin
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;
SHOW DATABASES LIKE 'amazon_admin';
EXIT;
```

## 4. 配置与脚本启动

### 4.1 生成本地配置文件

在项目根目录执行：

```powershell
Copy-Item .\server\config.local.example.yaml .\server\config.local.yaml
```

打开 `server\config.local.yaml`，确认或修改以下配置：

```yaml
system:
    env: local
    addr: 8888
    db-type: mysql
    oss-type: local
    use-redis: false

mysql:
    path: 127.0.0.1
    port: "3306"
    config: charset=utf8mb4&parseTime=True&loc=Local
    db-name: amazon_admin
    username: root
    password: "你的 MySQL root 密码"
    log-mode: error

local:
    path: uploads/file
    store-path: uploads/file
```

注意事项：

- `server\config.local.yaml` 是本地私密配置，不要提交到 Git。
- `use-redis: false` 是本地首次部署推荐值，不需要安装 Redis。
- `logistics` 节点里的示例路径如果是 macOS 路径，请清空或改成 Windows 本机路径，例如 `C:/workspace/rates/yuntu.xlsx`。
- Windows 路径建议写成 `C:/path/to/file.xlsx`，比反斜杠更不容易触发 YAML 转义问题。
- `amazon` 节点的 SP-API 凭据本地首次启动可以留空，授权、同步订单等外部接口功能需要后续再配置真实凭据。

### 4.2 低内存静态部署（2C2G 推荐）

该方式不会启动 `npm run dev` / Vite，也不会占用 `8080` 端口。运行时只有：

```text
Go 后端 8888 + web\dist 静态文件 + MySQL 8
```

如果你的 Windows 根目录就是 `C:\Users\Administrator\Desktop\eaon\system`，项目目录是 `test-eee-git`，MySQL root 密码使用 `123456a`，请用“管理员 PowerShell”直接执行：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
cd C:\Users\Administrator\Desktop\eaon\system\test-eee-git
.\scripts\win-lowmem-deploy.ps1
```

这个脚本会自动完成：

- 把 `PortableGit`、MySQL、Go、Node 临时加入当前 PowerShell 的 `PATH`。
- 如果 Go 或 Node 缺失，优先使用根目录下的 `go-installer*.msi`、`node-installer*.msi` 静默安装。
- 如果缺少 MySQL 依赖的 VC++ 运行库，优先使用根目录下的 `vc_redist.x64*.exe` 静默安装；找不到本地安装包时会尝试从 Microsoft 官方链接自动下载。
- 初始化 zip 版 MySQL 的 `data` 目录，启动 MySQL，并把 root 密码设置为 `123456a`。
- 创建 `amazon_admin` 数据库。
- 生成并修补 `server\config.local.yaml` 的 MySQL 配置，旧配置会备份到 `tmp\windows-lowmem-deploy`。
- 设置低内存构建参数：`STATIC_BUILD=1`、`SERVER_BUILD=1`、`NODE_OPTIONS=--max-old-space-size=1536`、`GO_BUILD_P=1`。
- 调用 `scripts\static-up.ps1` 构建并启动 Go 后端托管的静态站点。

如果目标机完全离线，不想执行 `git pull`，使用：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
cd C:\Users\Administrator\Desktop\eaon\system\test-eee-git
.\scripts\win-lowmem-deploy.ps1 -SkipGitPull
```

如果 MySQL root 密码不是 `123456a`，使用：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
cd C:\Users\Administrator\Desktop\eaon\system\test-eee-git
.\scripts\win-lowmem-deploy.ps1 -MySqlRootPassword "你的 MySQL root 密码"
```

如果前几次失败留下了半初始化的 MySQL `data` 目录，并且确认是首次部署、不要保留旧数据库，可以让脚本先备份旧目录再重新初始化：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
cd C:\Users\Administrator\Desktop\eaon\system\test-eee-git
.\scripts\win-lowmem-deploy.ps1 -SkipGitPull -ResetMySqlData
```

启动成功后打开：

```text
http://127.0.0.1:8888/#/login
```

默认登录：

```text
用户名：admin
密码：123456
```

关闭服务：

```powershell
cd C:\Users\Administrator\Desktop\eaon\system\test-eee-git
.\scripts\static-down.ps1
```

首次运行前，如果 MySQL root 密码不是脚本默认值 `123456a`，请先在当前 PowerShell 设置：

```powershell
$env:MYSQL_PASSWORD = "你的 MySQL root 密码"
$env:ADMIN_PASSWORD = "123456"
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
```

在项目根目录执行：

```powershell
.\scripts\static-up.ps1
```

脚本会自动完成：

- 检查 MySQL 是否可连接。
- 按需安装前端依赖。
- 执行 `npm run build:static` 生成 `web\dist`。
- 使用 `go build -p 1` 构建后端，降低构建峰值内存。
- 启动 Go 后端，并由 Go 后端直接托管 `web\dist`。
- 检查 `/health` 和前端首页。
- 如果数据库尚未初始化，自动调用 `/init/initdb` 初始化。

启动成功后访问：

```text
http://127.0.0.1:8888/#/login
```

关闭静态部署：

```powershell
.\scripts\static-down.ps1
```

脚本日志位置：

```text
tmp\static-runtime\logs\server.log
```

如果目标 Windows 机器只有 2G 内存，更稳的做法是在开发机先构建好 `web\dist` 和后端二进制，再在目标机复用构建产物：

```powershell
$env:STATIC_BUILD = "0"
$env:SERVER_BUILD = "0"
.\scripts\static-up.ps1
```

如果只想跳过前端构建，但仍在目标机编译 Go 后端：

```powershell
$env:STATIC_BUILD = "0"
.\scripts\static-up.ps1
```

可用环境变量：

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `API_PORT` | `8888` | 后端和静态页面共同使用的端口 |
| `MYSQL_HOST` | `127.0.0.1` | MySQL 地址 |
| `MYSQL_PORT` | `3306` | MySQL 端口 |
| `MYSQL_USER` | `root` | MySQL 用户 |
| `MYSQL_PASSWORD` | `123456a` 或配置文件中的密码 | MySQL 密码 |
| `MYSQL_DATABASE` | `amazon_admin` | 初始化数据库名 |
| `ADMIN_PASSWORD` | `123456` | 初始化后的后台管理员密码 |
| `STATIC_BUILD` | `1` | 是否执行 `npm run build:static`，设为 `0` 时复用现有 `web\dist` |
| `SERVER_BUILD` | `1` | 是否检查并构建 Go 二进制，设为 `0` 时优先复用现有 `tmp\static-runtime\bin\gva-server.exe` |
| `NODE_OPTIONS` | `--max-old-space-size=1536` | 前端构建时的 Node heap 上限 |
| `GO_BUILD_P` | `1` | Go 构建并发度，值越低构建越省内存但更慢 |

### 4.3 使用开发脚本启动和关闭（需要前端热更新时）

开发脚本参考 macOS 的 `scripts/dev-up.sh` 实现，会同时启动 Go 后端和 Vite 前端开发服务，适合本地开发和前端热更新，但内存占用高于静态部署。

开发脚本会自动完成：

- 复制 `server\config.local.example.yaml` 到 `server\config.local.yaml`，如果本地配置不存在。
- 检查 MySQL 是否可连接。
- 按需安装前端依赖。
- 按需构建后端二进制到 `tmp\dev-runtime\bin\gva-server.exe`。
- 启动后端和前端开发服务。
- 检查 `/health` 和前端首页。
- 如果数据库尚未初始化，自动调用 `/init/initdb` 初始化。
- 写入 PID 和日志到 `tmp\dev-runtime`。

首次运行前，如果 MySQL root 密码不是脚本默认值 `123456a`，请先在当前 PowerShell 设置：

```powershell
$env:MYSQL_PASSWORD = "你的 MySQL root 密码"
$env:ADMIN_PASSWORD = "123456"
```

如果你已经在 `server\config.local.yaml` 里配置了 `mysql.password`，也可以不设置 `MYSQL_PASSWORD`，脚本会优先读取本地配置里的密码。

在项目根目录执行：

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
.\scripts\dev-up.ps1
```

启动成功后会输出：

```text
Frontend: http://127.0.0.1:8080
Login: http://127.0.0.1:8080/#/login
Backend: http://127.0.0.1:8888
Health: http://127.0.0.1:8888/health
```

默认账号：

```text
用户名：admin
密码：123456
```

关闭本地开发环境：

```powershell
.\scripts\dev-down.ps1
```

如果需要改端口或数据库连接，可以使用环境变量：

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `API_PORT` | `8888` | 后端端口 |
| `WEB_PORT` | `8080` | 前端端口 |
| `WEB_HOST` | `127.0.0.1` | 前端监听地址 |
| `MYSQL_HOST` | `127.0.0.1` | MySQL 地址 |
| `MYSQL_PORT` | `3306` | MySQL 端口 |
| `MYSQL_USER` | `root` | MySQL 用户 |
| `MYSQL_PASSWORD` | `123456a` 或配置文件中的密码 | MySQL 密码 |
| `MYSQL_DATABASE` | `amazon_admin` | 初始化数据库名 |
| `ADMIN_PASSWORD` | `123456` | 初始化后的后台管理员密码 |

示例：

```powershell
$env:MYSQL_PASSWORD = "root密码"
$env:API_PORT = "8888"
$env:WEB_PORT = "8080"
.\scripts\dev-up.ps1
```

脚本日志位置：

```text
tmp\dev-runtime\logs\server.log
tmp\dev-runtime\logs\web.log
```

如果脚本失败，优先查看上述日志。

### 4.4 下载 Go 依赖（手动备用）

```powershell
cd C:\workspace\test-eee\server
go env -w GOPROXY=https://goproxy.cn,direct
go mod download
```

如果 Go 提示需要 toolchain，允许它下载，或确认本机 Go 版本不低于项目要求。

### 4.5 运行后端测试（手动备用）

```powershell
cd C:\workspace\test-eee\server
$env:GVA_CONFIG = "config.local.yaml"
go test ./...
```

测试通过时会看到多个 `ok` 或 `[no test files]`。

### 4.6 启动后端（手动备用）

保持在 `server` 目录：

```powershell
$env:GVA_CONFIG = "config.local.yaml"
go run .
```

后端启动后不要关闭这个 PowerShell 窗口。另开一个 PowerShell 验证健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:8888/health
```

预期输出：

```text
ok
```

Swagger 地址：

```text
http://127.0.0.1:8888/swagger/index.html
```

## 5. 初始化数据库

如果已经使用 `.\scripts\static-up.ps1` 或 `.\scripts\dev-up.ps1` 启动，脚本会自动检查并初始化数据库，通常不需要手动执行本节。

后端启动后，另开 PowerShell 执行：

```powershell
Invoke-RestMethod -Method Post http://127.0.0.1:8888/init/checkdb
```

如果返回数据里 `needInit` 为 `true`，执行初始化：

```powershell
$body = @{
  adminPassword = "123456"
  dbType = "mysql"
  host = "127.0.0.1"
  port = "3306"
  userName = "root"
  password = "你的 MySQL root 密码"
  dbName = "amazon_admin"
} | ConvertTo-Json

Invoke-RestMethod `
  -Method Post `
  -Uri http://127.0.0.1:8888/init/initdb `
  -ContentType "application/json" `
  -Body $body
```

初始化成功后，再检查一次：

```powershell
Invoke-RestMethod -Method Post http://127.0.0.1:8888/init/checkdb
```

预期 `needInit` 为 `false`。默认后台账号：

```text
用户名：admin
密码：123456
```

如果你在 `adminPassword` 使用了其他密码，登录时以你设置的密码为准。

## 6. 启动前端

如果已经使用 `.\scripts\static-up.ps1` 启动，前端已经由 Go 后端托管在 `http://127.0.0.1:8888/#/login`，不需要启动 Vite。

如果已经使用 `.\scripts\dev-up.ps1` 启动，脚本已经启动了前端开发服务，通常不需要手动执行本节。

另开一个 PowerShell，进入前端目录：

```powershell
cd C:\workspace\test-eee\web
npm install
```

先执行一次生产构建，确认前端代码和依赖完整：

```powershell
npm run build
```

如果只出现 chunk 体积较大的 warning，但最终显示构建完成，属于可接受结果。

启动开发服务：

```powershell
npm run dev:fast
```

前端默认地址：

```text
http://127.0.0.1:8080/#/login
```

前端开发配置位于 `web\.env.development`，默认使用：

```env
VITE_CLI_PORT = 8080
VITE_SERVER_PORT = 8888
VITE_BASE_API = /api
VITE_FILE_API = /api
VITE_BASE_PATH = http://127.0.0.1
```

正常情况下不需要修改。

## 7. 完整验收流程

按下面顺序检查，全部通过即代表 Windows 本地部署完成。

### 7.1 基础环境验收

```powershell
git --version
go version
node -v
npm -v
mysqladmin -h127.0.0.1 -P3306 -uroot -p ping
```

预期：所有工具输出正常，MySQL 返回 `mysqld is alive`。

### 7.2 后端验收

如果使用低内存静态部署，先执行：

```powershell
.\scripts\static-up.ps1
```

然后直接验证：

```powershell
Invoke-RestMethod http://127.0.0.1:8888/health
Invoke-RestMethod -Method Post http://127.0.0.1:8888/init/checkdb
```

如果使用开发模式脚本，也可以执行：

```powershell
.\scripts\dev-up.ps1
```

如果走手动流程，再执行下面的测试和启动命令。

```powershell
cd C:\workspace\test-eee\server
$env:GVA_CONFIG = "config.local.yaml"
go test ./...
```

启动后端：

```powershell
$env:GVA_CONFIG = "config.local.yaml"
go run .
```

另开 PowerShell：

```powershell
Invoke-RestMethod http://127.0.0.1:8888/health
Invoke-RestMethod -Method Post http://127.0.0.1:8888/init/checkdb
```

预期：

- `/health` 返回 `ok`。
- `/init/checkdb` 返回 `needInit=false`，或首次部署时先按第 5 节完成初始化。

### 7.3 前端验收

如果使用低内存静态部署，直接打开：

```text
http://127.0.0.1:8888/#/login
```

如果使用开发脚本启动，直接打开 `http://127.0.0.1:8080/#/login`。如果走手动流程，再执行：

```powershell
cd C:\workspace\test-eee\web
npm run build
npm run dev:fast
```

浏览器打开：

```text
http://127.0.0.1:8080/#/login
```

使用 `admin / 123456` 登录。登录后应进入后台首页。

### 7.4 Amazon 业务页验收

登录后抽查以下页面：

- 物流比价
- 店铺管理
- 订单管理
- 客服消息
- 财务看板

如果菜单未显示，先确认数据库已经初始化完成，并刷新浏览器页面。如果仍未显示，检查初始化接口是否执行成功，以及后端日志是否有菜单或权限初始化报错。

### 7.5 API 文档验收

浏览器打开：

```text
http://127.0.0.1:8888/swagger/index.html
```

能看到 Swagger 页面即为通过。

## 8. 常见问题

### 8.1 端口 8888 或 8080 被占用

查看占用进程：

```powershell
netstat -ano | findstr :8888
netstat -ano | findstr :8080
```

结束进程：

```powershell
taskkill /PID 进程ID /F
```

如果使用静态部署脚本启动，也可以直接执行：

```powershell
.\scripts\static-down.ps1
```

如果使用开发脚本启动，也可以直接执行：

```powershell
.\scripts\dev-down.ps1
```

也可以修改端口：

- 后端端口：`server\config.local.yaml` 的 `system.addr`
- 前端端口：`web\.env.development` 的 `VITE_CLI_PORT`
- 前端代理后端端口：`web\.env.development` 的 `VITE_SERVER_PORT`

### 8.2 MySQL 密码错误

错误通常表现为后端日志出现 `Access denied for user`。处理方式：

1. 用 `mysql -h127.0.0.1 -P3306 -uroot -p` 确认密码能登录。
2. 修改 `server\config.local.yaml` 的 `mysql.password`。
3. 重启后端。

### 8.3 `mysql.exe --version` 预检失败

新版 `scripts\win-lowmem-deploy.ps1` 会固定使用 `C:\Users\Administrator\Desktop\eaon\system\mysql-8.0.46-winx64\bin\mysql.exe`，并且不会因为 `mysql.exe --version` 返回非 0 直接停止。请重新拉取脚本后再执行：

```powershell
cd C:\Users\Administrator\Desktop\eaon\system\test-eee-git
.\scripts\win-lowmem-deploy.ps1
```

如果弹窗提示缺少 `VCRUNTIME140_1.dll`、`VCRUNTIME140.dll` 或 `MSVCP140.dll`，说明 Windows 缺少 Microsoft Visual C++ Redistributable x64。新版脚本会优先从 Microsoft 官方链接自动下载：

```text
https://aka.ms/vc14/vc_redist.x64.exe
```

如果目标机不能联网，请在其他机器下载 `vc_redist.x64.exe`，放到下面目录后重新执行脚本：

```text
C:\Users\Administrator\Desktop\eaon\system
```

脚本会自动静默安装该运行库。如果后续 MySQL 初始化或 `mysqladmin ping` 仍失败，手动检查：

```powershell
& C:\Users\Administrator\Desktop\eaon\system\mysql-8.0.46-winx64\bin\mysql.exe --version
& C:\Users\Administrator\Desktop\eaon\system\mysql-8.0.46-winx64\bin\mysqld.exe --version
& C:\Users\Administrator\Desktop\eaon\system\mysql-8.0.46-winx64\bin\mysqladmin.exe --host=127.0.0.1 --port=3306 --user=root ping
& C:\Users\Administrator\Desktop\eaon\system\mysql-8.0.46-winx64\bin\mysqladmin.exe --host=127.0.0.1 --port=3306 --user=root --password=123456a ping
```

如果手动安装运行库后仍弹同样错误，重启 Windows 后再运行部署脚本。

### 8.4 MySQL 启动失败或 3306 不可用

新版脚本会把 MySQL 初始化和启动日志写到：

```text
C:\Users\Administrator\Desktop\eaon\system\test-eee-git\tmp\windows-lowmem-deploy\mysql
```

失败后先看日志：

```powershell
Get-Content C:\Users\Administrator\Desktop\eaon\system\test-eee-git\tmp\windows-lowmem-deploy\mysql\mysqld.err.log -Tail 80
Get-Content C:\Users\Administrator\Desktop\eaon\system\test-eee-git\tmp\windows-lowmem-deploy\mysql\mysqld.out.log -Tail 80
```

检查 3306 是否被占用：

```powershell
netstat -ano | findstr :3306
```

如果确认占用者是前面部署脚本留下的 `mysqld.exe`，可以停止它后继续使用 3306：

```powershell
taskkill /PID 进程ID /F
.\scripts\win-lowmem-deploy.ps1 -SkipGitPull -ResetMySqlData
```

如果不想停止已有 MySQL，也可以改用 3307：

```powershell
.\scripts\win-lowmem-deploy.ps1 -SkipGitPull -ResetMySqlData -MySqlPort 3307
```

如果是首次部署，且前面失败留下了半初始化数据目录，可以备份旧目录并重建：

```powershell
.\scripts\win-lowmem-deploy.ps1 -SkipGitPull -ResetMySqlData
```

### 8.5 `GVA_CONFIG` 没有生效

使用 `.\scripts\dev-up.ps1` 启动时，脚本会自动设置 `GVA_CONFIG=config.local.yaml`。

必须在启动后端的同一个 PowerShell 窗口设置环境变量：

```powershell
cd C:\workspace\test-eee\server
$env:GVA_CONFIG = "config.local.yaml"
go run .
```

如果直接执行 `go run .`，后端会默认读取 `config.yaml`。

### 8.6 Go 依赖下载失败

设置代理后重试：

```powershell
go env -w GOPROXY=https://goproxy.cn,direct
go clean -modcache
go mod download
```

如果公司网络限制 HTTPS 访问，需要切换到可访问 Go module 的网络。

### 8.7 npm install 失败

清理后重装：

```powershell
cd C:\workspace\test-eee\web
Remove-Item -Recurse -Force .\node_modules -ErrorAction SilentlyContinue
npm cache verify
npm install
```

如果下载慢，可以设置 npm 镜像：

```powershell
npm config set registry https://registry.npmmirror.com
npm install
```

### 8.8 静态部署访问不了页面

按顺序检查：

1. `Invoke-RestMethod http://127.0.0.1:8888/health` 是否返回 `ok`。
2. `Test-Path .\web\dist\index.html` 是否返回 `True`。
3. 查看 `tmp\static-runtime\logs\server.log` 是否有 MySQL 或端口占用报错。
4. 如果改过 `server\config.local.yaml` 的 `system.addr`，要同步设置 `$env:API_PORT`。

### 8.9 前端登录时报接口错误

按顺序检查：

1. 后端窗口是否仍在运行。
2. `Invoke-RestMethod http://127.0.0.1:8888/health` 是否返回 `ok`。
3. 静态部署时，浏览器地址应为 `http://127.0.0.1:8888/#/login`。
4. 开发模式时，`web\.env.development` 的 `VITE_SERVER_PORT` 是否等于后端端口 `8888`。
5. 修改 `.env.development` 后必须重启 `npm run dev:fast`。

### 8.10 上传文件无法访问

确认 `server\config.local.yaml`：

```yaml
local:
    path: uploads/file
    store-path: uploads/file
```

并确保目录存在：

```powershell
cd C:\workspace\test-eee\server
New-Item -ItemType Directory -Force uploads\file
```

### 8.11 Windows 路径导致 YAML 报错

在 YAML 里优先使用正斜杠：

```yaml
logistics:
    yuntu-rate-file: C:/workspace/rates/yuntu.xlsx
```

不要写成未转义的单反斜杠路径：

```yaml
logistics:
    yuntu-rate-file: C:\workspace\rates\yuntu.xlsx
```

### 8.12 需要重新初始化数据库

如果只是本地测试环境，可以删除数据库后重新走第 5 节：

```powershell
mysql -h127.0.0.1 -P3306 -uroot -p
```

```sql
DROP DATABASE IF EXISTS amazon_admin;
CREATE DATABASE amazon_admin DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_unicode_ci;
EXIT;
```

然后重启后端并重新调用 `/init/initdb`。
