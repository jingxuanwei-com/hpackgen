# hpackgen  
**自动生成 HMCL 自动更新整合包配置的整合包构建工具**

hpackgen 是一个面向服务器管理员与整合包作者的自动化构建工具，用于生成 HMCL 自动更新整合包所需的全部配置文件。它能够自动扫描整合包内容、计算文件哈希、生成 manifest.json，并输出符合 HMCL 自动更新规范的整合包更新结构。

通过 hpackgen，你可以在每次更新模组、资源包或配置文件后，一键生成更新包并部署到你的更新服务器，让客户端 HMCL 自动检测并下载最新版本。

---

## ✦ 功能特点

- 自动生成 **manifest.json**（HMCL 自动更新整合包配置）
- 自动扫描整合包目录并生成 **files[] 文件列表与哈希**（SHA1）
- 自动写入 **整合包信息（name / author / version / description）**
- 自动生成 **addons[]（game / forge / neoforge 等版本信息）**
- 版本号支持 **模板占位符**（日期、时间、随机数、UUID），每次构建自动刷新
- 支持自定义 **fileApi**（整合包更新服务器地址）
- 自动监听 overrides 目录变动，增量更新 manifest
- 适合整合包频繁更新的服务器环境

---

## ✦ 快速开始

```bash
# 克隆项目
git clone https://github.com/你的用户名/hpackgen.git
cd hpackgen/1szt

# 运行（自动生成 .env 和初始 manifest）
go run ./main
```

首次运行会自动创建 `.env` 配置文件，编辑后重新运行即可生效。

---

## ✦ 配置说明（.env）

| 配置项 | 说明 |
|--------|------|
| `DATA_DIR` | 数据目录（默认 `data`，可改为 `.` 使用当前目录） |
| `MANIFEST_NAME` | 整合包名称 |
| `MANIFEST_AUTHOR` | 作者 |
| `MANIFEST_VERSION` | **版本号（支持模板占位符）** |
| `MANIFEST_DESCRIPTION` | 整合包描述（使用 `\n` 换行） |
| `MANIFEST_FILE_API` | 文件分发 API 地址 |
| `MANIFEST_ADDONS` | 游戏引擎和模组加载器版本（JSON 数组） |

### 版本号模板

`MANIFEST_VERSION` 支持以下占位符，运行时会自动替换：

| 占位符 | 说明 | 示例 |
|--------|------|------|
| `{date}` | 当前日期 | `20260714` |
| `{time}` | 当前时间 | `143052` |
| `{datetime}` | 日期+时间 | `20260714143052` |
| `{rand:N}` | N 位随机数 | `{rand:4}` → `8371` |
| `{shortuuid}` | 8 位随机十六进制 | `a3f1c9e2` |

**示例：**

```dotenv
MANIFEST_VERSION=1szt.{rand:9}          # 默认：1szt.837194620
MANIFEST_VERSION={date}.{rand:3}        # 20260714.831
MANIFEST_VERSION=v{date}-{shortuuid}   # v20260714-a3f1c9e2
MANIFEST_VERSION={datetime}             # 20260714143052
MANIFEST_VERSION=1.0.0                  # 固定版本（不含 {} 则不替换）
```

---

## ✦ 目录结构

```
hpackgen/
├── .env                      # 配置文件（自动生成）
├── data/
│   ├── overrides/            # 整合包文件（放入你的模组/资源包）
│   │   ├── mods/
│   │   ├── config/
│   │   └── ...
│   └── server-manifest.json  # 生成的 manifest（自动生成）
├── go.work
├── Dockerfile                # 容器镜像构建
└── .github/workflows/build.yml  # CI/CD 流水线
```

---

## ✦ 输出示例

```json
{
  "name": "1szt",
  "author": "1szt",
  "version": "1szt.837194620",
  "description": "# 欢迎来到 1szt 服务器\n\n感谢你选择加入我们的世界。\n交流与反馈请前往 QQ 群：565941634\n",
  "fileApi": "https://mc.1szt.com",
  "files": [
    {
      "path": "icon.png",
      "hash": "59ed14aa8d6276fc1ff3f40d3c681b828ec9a51a"
    }
  ],
  "addons": [
    {
      "id": "game",
      "version": "1.21.1"
    },
    {
      "id": "neoforge",
      "version": "21.1.236"
    }
  ]
}
```

---

## ✦ Docker

支持两种基础镜像，自动编译多架构（linux/amd64 + linux/arm64）：

| 基础镜像 | 大小 | 拉取命令 |
|---------|------|---------|
| Alpine（**默认**） | ~15MB | `docker pull ghcr.io/你的用户名/hpackgen:latest` |
| Debian slim | ~80MB | `docker pull ghcr.io/你的用户名/hpackgen:latest-debian` |

**镜像标签规则：**

| 触发方式 | Alpine 标签 | Debian 标签 |
|---------|------------|-------------|
| 推送代码 | `test` `test-alpine` `<sha>-alpine` | `test-debian` `<sha>-debian` |
| 发布/手动 | `latest` `latest-alpine` `<tag>` `<tag>-alpine` | `latest-debian` `<tag>-debian` |

本地构建：

```bash
docker buildx build --target=debian --platform linux/amd64,linux/arm64 -t hpackgen:debian .
docker buildx build --target=alpine --platform linux/amd64,linux/arm64 -t hpackgen:alpine .
```

---

## ✦ GitHub Actions 自动构建

推送代码到 GitHub 后自动触发 CI/CD 流水线：

| 触发方式 | 操作 |
|---------|------|
| **🔄 自动构建** | 推送代码到任意分支 |
| **🏷️ 发布构建** | 创建 GitHub Release |
| **👆 手动构建** | Actions 页面点击 `Run workflow` |

### 构建产物

每次构建自动生成：

- **6 个架构的二进制文件** — Linux / Windows / macOS × amd64 / arm64
- **2 种容器镜像** — Debian slim / Alpine，均支持 amd64 + arm64 多架构
- **Release 资产** — 自动上传到 Release 页面（含 SHA256 校验和）

二进制文件下载后在终端直接运行：

```bash
# Linux / macOS
chmod +x hpackgen-linux-amd64
./hpackgen-linux-amd64

# Windows
hpackgen-win-amd64.exe
```

> 首次使用前需在 GitHub 仓库 `Settings → Actions → General` 中，将 **Workflow permissions** 设为 **Read and write permissions**。

---

## ✦ 适用场景

- 需要频繁更新模组或配置的服务器
- 需要自动分发整合包更新的 HMCL 用户
- 想要减少手工编辑 manifest.json 的整合包作者
- 想要构建自动化整合包发布流水线的开发者

---

## ✦ 许可

[Apache-2.0 license](https://github.com/jingxuanwei-com/hpackgen#Apache-2.0-1-ov-file)
