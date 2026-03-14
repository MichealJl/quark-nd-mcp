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
List files and folders in a directory.

Parameters:
- `parent_id` (string): Parent folder ID. Use "0" for root directory.

### create_folder
Create a new folder.

Parameters:
- `parent_id` (string): Parent folder ID. Use "0" for root.
- `folder_name` (string): Name of the new folder.

### rename
Rename a file or folder.

Parameters:
- `file_id` (string): File or folder ID to rename.
- `new_name` (string): New name.

### delete
Delete files or folders.

Parameters:
- `file_ids` (array of strings): List of file/folder IDs to delete.

### move
Move files or folders.

Parameters:
- `file_ids` (array of strings): List of file/folder IDs to move.
- `dest_id` (string): Destination folder ID.

### copy
Copy files or folders.

Parameters:
- `file_ids` (array of strings): List of file/folder IDs to copy.
- `dest_id` (string): Destination folder ID.

### get_info
Get detailed information about a file or folder.

Parameters:
- `file_id` (string): File or folder ID.

### get_download_url
Get download URL for a file.

Parameters:
- `file_id` (string): File ID.

### upload_file
Upload a local file.

Parameters:
- `parent_id` (string): Parent folder ID. Use "0" for root.
- `file_path` (string): Local file path to upload.

### regex_rename
Batch rename files using regex.

Parameters:
- `parent_id` (string): Parent folder ID containing files to rename.
- `pattern` (string): Regular expression pattern.
- `replacement` (string): Replacement string.

## License

MIT License