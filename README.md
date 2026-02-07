# mindmapgen

从缩进文本或 Mermaid 思维导图语法生成 PNG 思维导图。支持 CLI、HTTP API 和 MCP 工具调用。

## CLI

从文件生成 PNG：

```sh
go run ./cmd/mindmapgen -i examples/map.txt -o artifacts/map.png
```

从原始文本生成：

```sh
go run ./cmd/mindmapgen -raw "mindmap\n  root((Main Topic))\n    Subtopic" -o artifacts/map.png
```

选择主题和布局：

```sh
go run ./cmd/mindmapgen -i examples/map.txt -o artifacts/map.png -theme dark -layout both
```

布局选项：`right`（默认）、`left`、`both`。

## HTTP API

生成 PNG：

```sh
curl -X POST "http://localhost:8080/api/gen?media=raw&theme=default&layout=both" \
  -H "Content-Type: text/plain" \
  --data-binary $'mindmap\n  root((Main Topic))\n    Subtopic'
```

列出主题：

```sh
curl "http://localhost:8080/api/themes"
```

## MCP

工具名：`generate_mindmap`

参数：
- `content`（string，必填）
- `theme`（string，可选）
- `layout`（string，可选：`right`、`left`、`both`）

### Stdio 与 Streamable HTTP 如何选择

| 场景 | 推荐传输 | 原因 |
|------|---------|------|
| Claude Desktop / Claude Code | **stdio** | 宿主直接管理进程生命周期，零网络配置 |
| Cursor / Windsurf 等 IDE | **stdio** | 主流 IDE MCP 插件均优先支持 stdio |
| 多用户共享 / 远程部署 | **Streamable HTTP** | 一个服务端实例服务多个客户端，适合团队/服务器场景 |
| Docker / K8s 部署 | **Streamable HTTP** | 容器化场景天然适合网络传输 |

对大多数个人用户，推荐 **stdio**；对团队/服务端部署，推荐 **Streamable HTTP**。

### 安装

```bash
# stdio 传输（推荐大多数用户使用）
go install github.com/hellodeveye/mindmapgen/cmd/mcp-stdio@latest

# Streamable HTTP 传输（适用于服务端/团队部署）
go install github.com/hellodeveye/mindmapgen/cmd/mcp-server@latest
```

### Claude Desktop 配置（stdio）

在 Claude Desktop 配置文件中添加：

```json
{
  "mcpServers": {
    "mindmapgen": {
      "command": "mcp-stdio",
      "env": {}
    }
  }
}
```

配置文件路径：
- macOS：`~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows：`%APPDATA%\Claude\claude_desktop_config.json`

### Claude Code 配置（stdio）

通过命令行添加：

```bash
claude mcp add mindmapgen mcp-stdio
```

或在 Claude Code MCP 配置中添加：

```json
{
  "mcpServers": {
    "mindmapgen": {
      "command": "mcp-stdio"
    }
  }
}
```

### Streamable HTTP

启动服务：

```bash
mcp-server -addr :8082
```

客户端连接端点：`http://localhost:8082/mcp`

### R2 存储（可选）

未配置 R2 时，生成的图片以 base64 编码返回——无需额外配置即可正常使用。

如需额外获取图片 URL，请配置 Cloudflare R2：

```bash
export R2_ACCOUNT_ID="your-account-id"
export R2_ACCESS_KEY_ID="your-access-key"
export R2_ACCESS_KEY_SECRET="your-secret-key"
export R2_BUCKET_NAME="your-bucket"
export R2_DOMAIN="your-r2-domain"
```

配置 R2 后，工具响应将同时包含 base64 图片和公开访问的 URL。
