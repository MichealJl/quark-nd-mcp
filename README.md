# 夸克网盘 MCP Server

基于 Go 语言实现的夸克网盘 Model Context Protocol (MCP) 服务器。

## 功能特性

- **列出文件/文件夹** - 列出目录内容
- **创建文件夹** - 创建新文件夹
- **重命名** - 重命名文件和文件夹
- **删除** - 删除文件和文件夹
- **移动** - 移动文件和文件夹
- **复制** - 复制文件和文件夹
- **获取信息** - 获取文件或文件夹的详细信息
- **下载文件** - 异步下载，支持并发连接（3线程，10MB分片）
- **上传文件** - 上传本地文件到夸克网盘
- **正则重命名** - 使用正则表达式批量重命名文件
- **搜索** - 根据关键词递归搜索文件
- **获取分享文件列表** - 获取分享链接根目录的文件列表
- **获取分享详情** - 获取分享链接中指定文件夹的文件列表
- **保存分享文件** - 将分享链接中的文件保存到自己的网盘

## 安装

### 下载预编译二进制文件

从 GitHub Releases 下载适合您平台的最新版本：

| 平台 | 架构 | 下载链接 |
|------|------|----------|
| macOS | Apple Silicon (M1/M2/M3/M4) | [quark-nd-mcp_darwin_apple_silicon.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases) |
| macOS | Intel | [quark-nd-mcp_darwin_intel.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases) |
| Linux | x64 | [quark-nd-mcp_linux_amd64.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases) |
| Linux | ARM64 | [quark-nd-mcp_linux_arm64.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases) |
| Windows | x64 | [quark-nd-mcp_windows_amd64.zip](https://github.com/MichealJl/quark-nd-mcp/releases) |
| Windows | ARM64 | [quark-nd-mcp_windows_arm64.zip](https://github.com/MichealJl/quark-nd-mcp/releases) |

### 从源码编译

```bash
go build -o quark-nd-mcp .
```

## 配置

在 `~/.quark-nd-disk/config.json` 创建配置文件：

```json
{
  "cookie": "你的夸克网盘cookie"
}
```

### 如何获取 Cookie

1. 在浏览器中登录 [夸克网盘](https://pan.quark.cn)
2. 打开开发者工具（F12）
3. 切换到 Network（网络）标签
4. 刷新页面
5. 找到任意请求到 `drive.quark.cn` 的请求
6. 从请求头中复制 `Cookie` 的值

## 使用方法

### 运行 MCP 服务器

```bash
./quark-nd-mcp
```

或使用自定义配置路径：

```bash
./quark-nd-mcp -config /path/to/config.json
```

### 配合 Claude Desktop 使用

在 Claude Desktop 配置文件中添加（macOS 路径：`~/Library/Application Support/Claude/claude_desktop_config.json`）：

```json
{
  "mcpServers": {
    "quark-nd-mcp": {
      "command": "/path/to/quark-nd-mcp"
    }
  }
}
```

## 可用工具

### list_files（列出文件）
列出指定目录下的所有文件和文件夹。

参数：
- `path`（字符串）：目录路径。使用 `/` 或空字符串表示根目录。例如：`/我的文档`

### create_folder（创建文件夹）
在指定路径创建新文件夹。

参数：
- `path`（字符串）：新文件夹的完整路径。例如：`/父文件夹/新文件夹` 或 `新文件夹`（在根目录创建）

### rename（重命名）
重命名文件或文件夹。

参数：
- `old_path`（字符串）：文件或文件夹的当前路径。例如：`/文件夹/旧名称`
- `new_name`（字符串）：新名称（不是完整路径）。例如：`新名称`

### delete（删除）
删除指定路径的文件或文件夹。

参数：
- `paths`（字符串数组）：要删除的路径列表。例如：`["/文件夹/文件.txt", "/文件夹/子文件夹"]`

### move（移动）
移动文件或文件夹到目标目录。

参数：
- `source_paths`（字符串数组）：要移动的源路径列表。例如：`["/文件夹/文件.txt"]`
- `dest_path`（字符串）：目标文件夹路径。使用 `/` 表示根目录。例如：`/目标目录`

### copy（复制）
复制文件或文件夹到目标目录。

参数：
- `source_paths`（字符串数组）：要复制的源路径列表。例如：`["/文件夹/文件.txt"]`
- `dest_path`（字符串）：目标文件夹路径。使用 `/` 表示根目录。例如：`/目标目录`

### get_info（获取信息）
获取文件或文件夹的详细信息。

参数：
- `path`（字符串）：文件或文件夹的路径。例如：`/文件夹/文件.txt`

### download_file（下载文件）
启动异步下载任务，将夸克网盘文件下载到本地。返回任务ID用于跟踪进度。

参数：
- `source_path`（字符串）：夸克网盘中的文件路径。例如：`/文件夹/文件.txt`
- `local_path`（字符串）：本地保存路径。例如：`/Users/name/Downloads/文件.txt`

返回：
- `task_id`（字符串）：用于 `get_download_status` 查询进度
- `file_size`（int64）：文件总大小（字节）

### get_download_status（获取下载状态）
查询下载任务的状态和进度。

参数：
- `task_id`（字符串）：`download_file` 返回的下载任务ID。例如：`dl_1234567890`

返回：
- `status`（字符串）：`pending`（等待）、`running`（运行中）、`completed`（已完成）、`failed`（失败）或 `canceled`（已取消）
- `progress`（字符串）：进度百分比。例如：`45.23%`
- `downloaded`（int64）：已下载字节数
- `total_size`（int64）：文件总大小（字节）
- `error`（字符串）：失败时的错误信息

### cancel_download（取消下载）
取消正在运行的下载任务。

参数：
- `task_id`（字符串）：要取消的下载任务ID。例如：`dl_1234567890`

### list_downloads（列出下载任务）
列出所有下载任务及其状态信息。

返回下载任务对象数组，包含状态、进度和文件信息。

### upload_file（上传文件）
将本地文件上传到夸克网盘。

参数：
- `dest_path`（字符串）：目标文件夹路径。使用 `/` 表示根目录。例如：`/我的文档`
- `local_path`（字符串）：本地文件路径。例如：`/Users/name/Downloads/文件.txt`

### regex_rename（正则重命名）
使用正则表达式批量重命名文件夹内的文件。

参数：
- `path`（字符串）：包含要重命名文件的目录路径。例如：`/photos`
- `pattern`（字符串）：匹配文件名的正则表达式模式。例如：`IMG_(\d+)`
- `replacement`（字符串）：替换字符串。使用 `$1`、`$2` 表示捕获组。例如：`Photo_$1`

### search（搜索）
在指定目录下递归搜索文件或文件夹。

参数：
- `path`（字符串）：搜索的目录路径。使用 `/` 表示根目录。例如：`/我的文档`
- `keyword`（字符串）：搜索关键词，匹配文件/文件夹名称

### get_share_file_list（获取分享文件列表）
获取夸克分享链接根目录的文件列表。

参数：
- `share_url`（字符串）：夸克分享链接。例如：`https://pan.quark.cn/s/abc123` 或带密码 `https://pan.quark.cn/s/abc123?pwd=xyz`

返回：
- 文件列表，包含 id、name、size、is_folder、category、时间戳等信息

### get_share_detail（获取分享详情）
获取分享链接中指定文件夹内的文件列表。

参数：
- `share_url`（字符串）：夸克分享链接。例如：`https://pan.quark.cn/s/abc123`
- `folder_id`（字符串）：文件夹ID（fid）。使用 `0` 表示根目录，或传入文件夹的 fid 浏览子目录

返回：
- 文件列表，包含 id、name、size、is_folder、category、时间戳等信息

### save_from_share（保存分享文件）
将分享链接中的文件保存到自己的夸克网盘。

参数：
- `share_url`（字符串）：夸克分享链接。例如：`https://pan.quark.cn/s/abc123`
- `folder_id`（字符串，可选）：文件所在的文件夹ID。使用 `0` 表示根目录（默认）。如果要保存子目录中的文件，请先通过 `get_share_detail` 获取文件夹的 fid。例如：`abc123def456`
- `dest_path`（字符串）：目标保存路径。使用 `/` 表示根目录。例如：`/来自分享`
- `file_ids`（字符串数组，可选）：要保存的文件ID列表。留空则保存该文件夹下全部文件。例如：`["file_id_1", "file_id_2"]`

返回：
- `task_id`（字符串）：保存任务ID
- `saved_count`（int）：已保存文件数量
- `saved_files`（字符串数组）：已保存的文件名列表
- `folder_id`（字符串）：源文件夹ID

## 下载流程示例

```
1. download_file(source_path="/视频/电影.mp4", local_path="/tmp/电影.mp4")
   → 返回：{ "task_id": "dl_1709876543210", "file_size": 1500000000 }

2. get_download_status(task_id="dl_1709876543210")
   → 返回：{ "status": "running", "progress": "35.50%", "downloaded": 532500000 }

3. get_download_status(task_id="dl_1709876543210")
   → 返回：{ "status": "completed", "progress": "100.00%" }
```

## 分享文件操作示例

### 保存根目录全部文件
```
save_from_share(share_url="https://pan.quark.cn/s/abc123", dest_path="/来自分享")
→ 返回：{ "saved_count": 5, "saved_files": ["文件1.mp4", "文件夹A", ...] }
```

### 保存根目录特定文件
```
1. get_share_file_list(share_url="https://pan.quark.cn/s/abc123")
   → 返回：[{ "id": "xxx", "name": "文件.mp4", "is_folder": false }, ...]

2. save_from_share(share_url="https://pan.quark.cn/s/abc123", dest_path="/来自分享", file_ids=["xxx"])
   → 返回：{ "saved_count": 1, "saved_files": ["文件.mp4"] }
```

### 保存子目录中的特定文件
```
1. get_share_file_list(share_url="https://pan.quark.cn/s/abc123")
   → 返回：[{ "id": "folder_abc", "name": "文件夹A", "is_folder": true }, ...]

2. get_share_detail(share_url="https://pan.quark.cn/s/abc123", folder_id="folder_abc")
   → 返回：[{ "id": "file_xyz", "name": "电影.mp4", "is_folder": false }, ...]

3. save_from_share(share_url="https://pan.quark.cn/s/abc123", folder_id="folder_abc", file_ids=["file_xyz"], dest_path="/来自分享")
   → 返回：{ "saved_count": 1, "saved_files": ["电影.mp4"], "folder_id": "folder_abc" }
```

## 许可证

MIT license