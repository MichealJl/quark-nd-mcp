# Quark Net Disk MCP Server

A Model Context Protocol (MCP) server for Quark Cloud Drive (夸克网盘), implemented in Go.

## Features

- **List files/folders** - List contents of a directory
- **Create folder** - Create new folders
- **Rename** - Rename files and folders
- **Delete** - Delete files and folders
- **Move** - Move files and folders
- **Copy** - Copy files and folders
- **Get info** - Get detailed information about a file or folder
- **Get download URL** - Get download link for files
- **Upload file** - Upload local files to Quark drive
- **Regex rename** - Batch rename files using regular expressions

## Installation

### Download Pre-built Binaries

Download the latest release for your platform:

| Platform | Architecture | Download |
|----------|-------------|----------|
| macOS | Apple Silicon (M1/M2/M3/M4) | [quark-nd-mcp_1.0.0_darwin_apple_silicon.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases/download/v1.0.0/quark-nd-mcp_1.0.0_darwin_apple_silicon.tar.gz) |
| macOS | Intel | [quark-nd-mcp_1.0.0_darwin_intel.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases/download/v1.0.0/quark-nd-mcp_1.0.0_darwin_intel.tar.gz) |
| Linux | x64 | [quark-nd-mcp_1.0.0_linux_amd64.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases/download/v1.0.0/quark-nd-mcp_1.0.0_linux_amd64.tar.gz) |
| Linux | ARM64 | [quark-nd-mcp_1.0.0_linux_arm64.tar.gz](https://github.com/MichealJl/quark-nd-mcp/releases/download/v1.0.0/quark-nd-mcp_1.0.0_linux_arm64.tar.gz) |
| Windows | x64 | [quark-nd-mcp_1.0.0_windows_amd64.zip](https://github.com/MichealJl/quark-nd-mcp/releases/download/v1.0.0/quark-nd-mcp_1.0.0_windows_amd64.zip) |
| Windows | ARM64 | [quark-nd-mcp_1.0.0_windows_arm64.zip](https://github.com/MichealJl/quark-nd-mcp/releases/download/v1.0.0/quark-nd-mcp_1.0.0_windows_arm64.zip) |

### Build from Source

```bash
go build -o quark-nd-mcp .
```

## Configuration

Create a config file at `~/.quark-nd-disk/config.json`:

```json
{
  "cookie": "your_quark_cookie_here"
}
```

### How to get your cookie

1. Login to [Quark Drive](https://pan.quark.cn) in your browser
2. Open Developer Tools (F12)
3. Go to Network tab
4. Refresh the page
5. Find any request to `drive.quark.cn`
6. Copy the `Cookie` header value from the request

## Usage

### Running the MCP Server

```bash
./quark-nd-mcp
```

Or with a custom config path:

```bash
./quark-nd-mcp -config /path/to/config.json
```

### Using with Claude Desktop

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "quark-nd-mcp": {
      "command": "/path/to/quark-nd-mcp"
    }
  }
}
```

## Available Tools

### list_files
List files and folders in a directory by path.

Parameters:
- `path` (string): Directory path to list. Use `/` or empty string for root directory. Example: `/我的文档`

### create_folder
Create a new folder at the specified path.

Parameters:
- `path` (string): Full path for the new folder. Example: `/parent_folder/new_folder` or `new_folder` for root

### rename
Rename a file or folder by path.

Parameters:
- `old_path` (string): Current path of the file or folder. Example: `/folder/old_name`
- `new_name` (string): New name (not full path). Example: `new_name`

### delete
Delete files or folders by path.

Parameters:
- `paths` (array of strings): List of paths to delete. Example: `["/folder/file.txt", "/folder/subfolder"]`

### move
Move files or folders to a destination directory.

Parameters:
- `source_paths` (array of strings): List of source paths to move. Example: `["/folder/file.txt"]`
- `dest_path` (string): Destination folder path. Use `/` for root. Example: `/destination`

### copy
Copy files or folders to a destination directory.

Parameters:
- `source_paths` (array of strings): List of source paths to copy. Example: `["/folder/file.txt"]`
- `dest_path` (string): Destination folder path. Use `/` for root. Example: `/destination`

### get_info
Get detailed information about a file or folder by path.

Parameters:
- `path` (string): Path to file or folder. Example: `/folder/file.txt`

### get_download_url
Get the download URL for a file by path.

Parameters:
- `path` (string): Path to the file. Example: `/folder/file.txt`

### upload_file
Upload a local file to Quark drive.

Parameters:
- `dest_path` (string): Destination folder path. Use `/` for root. Example: `/我的文档`
- `local_path` (string): Local file path to upload. Example: `/Users/name/Downloads/file.txt`

### regex_rename
Batch rename files in a directory using regular expression pattern.

Parameters:
- `path` (string): Directory path containing files to rename. Example: `/photos`
- `pattern` (string): Regular expression pattern to match file names. Example: `IMG_(\d+)`
- `replacement` (string): Replacement string. Use `$1`, `$2` for captured groups. Example: `Photo_$1`

### search
Search for files or folders by name in a directory (recursively).

Parameters:
- `path` (string): Directory path to search in. Use `/` for root. Example: `/我的文档`
- `keyword` (string): Keyword to search for in file/folder names

## License

MIT License