package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MichealJl/quark-nd-mcp/config"
	"github.com/MichealJl/quark-nd-mcp/quark"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server represents the MCP server for Quark drive
type Server struct {
	client      *quark.Client
	server      *mcp.Server
	downloadMgr *quark.DownloadManager
}

// NewServer creates a new MCP server
func NewServer(cfg *config.Config) *Server {
	client := quark.NewClient(cfg.Cookie)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "quark-nd-mcp",
		Version: "1.0.0",
	}, nil)

	s := &Server{
		client:      client,
		server:      server,
		downloadMgr: quark.NewDownloadManager(client),
	}

	// Register all tools
	s.registerTools()

	return s
}

// Run starts the MCP server using stdio transport
func (s *Server) Run(ctx context.Context) error {
	return s.server.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) registerTools() {
	// Tool: List files
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list_files",
		Description: "列出指定目录下的所有文件和文件夹。使用 '/' 或空字符串表示根目录。",
	}, s.listFiles)

	// Tool: Create folder
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "create_folder",
		Description: "在指定路径创建新文件夹。例如：'/父文件夹/新文件夹' 或直接使用 '新文件夹' 在根目录创建。",
	}, s.createFolder)

	// Tool: Rename
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "rename",
		Description: "重命名文件或文件夹。需要提供原路径和新名称。例如：'/文件夹/旧名称' 重命名为 '新名称'",
	}, s.rename)

	// Tool: Delete
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "delete",
		Description: "删除指定路径的文件或文件夹。支持批量删除，传入路径数组即可。例如：['/文件夹/文件.txt', '/文件夹/子文件夹']",
	}, s.delete)

	// Tool: Move
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "move",
		Description: "移动文件或文件夹到目标目录。使用 '/' 表示根目录。例如：将 '/源/文件.txt' 移动到 '/目标目录/'",
	}, s.move)

	// Tool: Copy
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "copy",
		Description: "复制文件或文件夹到目标目录。使用 '/' 表示根目录。例如：将 '/源/文件.txt' 复制到 '/目标目录/'",
	}, s.copy)

	// Tool: Get info
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_info",
		Description: "获取文件或文件夹的详细信息，包括名称、大小、类型、创建和更新时间。例如：'/文件夹/文件.txt'",
	}, s.getInfo)

	// Tool: Download file (async)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "download_file",
		Description: "启动异步下载任务，将夸克网盘文件下载到本地。返回任务ID，可使用 get_download_status 查询进度。支持并发下载（3线程，10MB分片）。",
	}, s.downloadFile)

	// Tool: Get download status
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_download_status",
		Description: "根据任务ID查询下载任务的状态和进度。返回状态（pending/running/completed/failed/canceled）、进度百分比、已下载字节等信息。",
	}, s.getDownloadStatus)

	// Tool: Cancel download
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cancel_download",
		Description: "取消正在运行的下载任务。需要提供任务ID。",
	}, s.cancelDownload)

	// Tool: List downloads
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list_downloads",
		Description: "列出所有下载任务及其状态信息。",
	}, s.listDownloads)

	// Tool: Upload file
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "upload_file",
		Description: "将本地文件上传到夸克网盘。需要指定目标文件夹路径（使用 '/' 表示根目录）和本地文件路径。",
	}, s.uploadFile)

	// Tool: Regex rename
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "regex_rename",
		Description: "使用正则表达式批量重命名文件夹内的文件。例如：path='/photos', pattern='IMG_(.*)', replacement='Photo_$1'",
	}, s.regexRename)

	// Tool: Search
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "search",
		Description: "在指定目录下递归搜索文件或文件夹。根据关键词匹配文件名。例如：path='/文件夹', keyword='photo'",
	}, s.search)

	// Tool: Get share file list
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_share_file_list",
		Description: "获取夸克分享链接根目录的文件列表。返回分享中所有文件和文件夹的信息。例如：share_url='https://pan.quark.cn/s/abc123' 或带密码 'https://pan.quark.cn/s/abc123?pwd=xyz'",
	}, s.getShareFileList)

	// Tool: Get share detail
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_share_detail",
		Description: "获取分享链接中指定文件夹内的文件列表。使用 folder_id='0' 表示根目录，或传入文件夹的 fid 浏览子目录。例如：share_url='https://pan.quark.cn/s/abc123', folder_id='文件夹fid'",
	}, s.getShareDetail)

	// Tool: Save from share
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "save_from_share",
		Description: "将分享链接中的文件保存到自己的夸克网盘。需要提供分享链接、目标保存路径。可通过 folder_id 指定文件所在的文件夹（默认为根目录），可选指定要保存的文件ID列表（留空则保存该文件夹下全部文件）。例如：保存根目录所有文件 share_url='https://pan.quark.cn/s/abc123', dest_path='/来自分享'；保存子目录特定文件 share_url='https://pan.quark.cn/s/abc123', folder_id='文件夹fid', file_ids=['文件fid'], dest_path='/来自分享'",
	}, s.saveFromShare)
}

// --- Tool Input Types ---

type ListFilesInput struct {
	Path string `json:"path" jsonschema:"Directory path to list. Use '/' or '' for root directory. Example: '/我的文档'"`
}

type CreateFolderInput struct {
	Path string `json:"path" jsonschema:"Full path for the new folder. Example: '/parent_folder/new_folder' or 'new_folder' for root"`
}

type RenameInput struct {
	OldPath string `json:"old_path" jsonschema:"Current path of the file or folder. Example: '/folder/old_name'"`
	NewName string `json:"new_name" jsonschema:"New name (not full path). Example: 'new_name'"`
}

type DeleteInput struct {
	Paths []string `json:"paths" jsonschema:"List of paths to delete. Example: ['/folder/file.txt', '/folder/subfolder']"`
}

type MoveInput struct {
	SourcePaths []string `json:"source_paths" jsonschema:"List of source paths to move. Example: ['/folder/file.txt']"`
	DestPath    string   `json:"dest_path" jsonschema:"Destination folder path. Use '/' for root. Example: '/destination'"`
}

type CopyInput struct {
	SourcePaths []string `json:"source_paths" jsonschema:"List of source paths to copy. Example: ['/folder/file.txt']"`
	DestPath    string   `json:"dest_path" jsonschema:"Destination folder path. Use '/' for root. Example: '/destination'"`
}

type GetInfoInput struct {
	Path string `json:"path" jsonschema:"Path to file or folder. Example: '/folder/file.txt'"`
}

type DownloadFileInput struct {
	SourcePath string `json:"source_path" jsonschema:"Path to the file in Quark drive. Example: '/folder/file.txt'"`
	LocalPath  string `json:"local_path" jsonschema:"Local file path to save. Example: '/Users/name/Downloads/file.txt'"`
}

type GetDownloadStatusInput struct {
	TaskID string `json:"task_id" jsonschema:"Download task ID. Example: 'dl_1234567890'"`
}

type CancelDownloadInput struct {
	TaskID string `json:"task_id" jsonschema:"Download task ID to cancel. Example: 'dl_1234567890'"`
}

type UploadFileInput struct {
	DestPath  string `json:"dest_path" jsonschema:"Destination folder path. Use '/' for root. Example: '/我的文档'"`
	LocalPath string `json:"local_path" jsonschema:"Local file path to upload. Example: '/Users/name/Downloads/file.txt'"`
}

type RegexRenameInput struct {
	Path        string `json:"path" jsonschema:"Directory path containing files to rename. Example: '/photos'"`
	Pattern     string `json:"pattern" jsonschema:"Regular expression pattern to match file names. Example: 'IMG_(\\d+)'"`
	Replacement string `json:"replacement" jsonschema:"Replacement string. Use $1, $2 for captured groups. Example: 'Photo_$1'"`
}

type SearchInput struct {
	Path    string `json:"path" jsonschema:"Directory path to search in. Use '/' for root. Example: '/我的文档'"`
	Keyword string `json:"keyword" jsonschema:"Keyword to search for in file/folder names"`
}

type GetShareFileListInput struct {
	ShareURL string `json:"share_url" jsonschema:"Quark share URL. Example: 'https://pan.quark.cn/s/abc123' or 'https://pan.quark.cn/s/abc123?pwd=xyz'"`
}

type GetShareDetailInput struct {
	ShareURL string `json:"share_url" jsonschema:"Quark share URL. Example: 'https://pan.quark.cn/s/abc123'"`
	FolderID string `json:"folder_id" jsonschema:"Folder ID (fid) to list files from. Use '0' for root directory. Example: 'abc123def456'"`
}

type SaveFromShareInput struct {
	ShareURL string   `json:"share_url" jsonschema:"夸克分享链接。例如：'https://pan.quark.cn/s/abc123'"`
	FolderID string   `json:"folder_id" jsonschema:"文件所在的文件夹ID。使用 '0' 表示根目录。如果要保存子目录中的文件，请先通过 get_share_detail 获取文件夹的 fid。例如：'abc123def456'"`
	DestPath string   `json:"dest_path" jsonschema:"目标保存路径。使用 '/' 表示根目录。例如：'/来自分享'"`
	FileIDs  []string `json:"file_ids" jsonschema:"要保存的文件ID列表。留空则保存该文件夹下的全部文件。例如：['file_id_1', 'file_id_2']"`
}

// --- Tool Handlers ---

func (s *Server) listFiles(ctx context.Context, req *mcp.CallToolRequest, args ListFilesInput) (*mcp.CallToolResult, any, error) {
	folderID, err := s.client.ResolvePath(ctx, args.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", args.Path, err)
	}

	files, err := s.client.ListFiles(ctx, folderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list files: %w", err)
	}

	result := make([]map[string]interface{}, len(files))
	for i, f := range files {
		result[i] = map[string]interface{}{
			"name":       f.Name,
			"size":       f.Size,
			"is_folder":  f.IsFolder,
			"updated_at": f.UpdatedAt.Format("2006-01-02 15:04:05"),
			"created_at": f.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) createFolder(ctx context.Context, req *mcp.CallToolRequest, args CreateFolderInput) (*mcp.CallToolResult, any, error) {
	parentID, folderName, err := s.client.ResolveParentPathAndName(ctx, args.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", args.Path, err)
	}

	fid, err := s.client.CreateFolder(ctx, parentID, folderName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create folder: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"fid":     fid,
		"message": fmt.Sprintf("Folder '%s' created successfully at '%s'", folderName, args.Path),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) rename(ctx context.Context, req *mcp.CallToolRequest, args RenameInput) (*mcp.CallToolResult, any, error) {
	fileID, _, _, err := s.client.ResolvePathToFileOrFolder(ctx, args.OldPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", args.OldPath, err)
	}

	err = s.client.Rename(ctx, fileID, args.NewName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to rename: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Renamed to '%s' successfully", args.NewName),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) delete(ctx context.Context, req *mcp.CallToolRequest, args DeleteInput) (*mcp.CallToolResult, any, error) {
	fids := make([]string, 0, len(args.Paths))
	for _, path := range args.Paths {
		fileID, _, _, err := s.client.ResolvePathToFileOrFolder(ctx, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", path, err)
		}
		fids = append(fids, fileID)
	}

	err := s.client.Delete(ctx, fids)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to delete: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deleted %d item(s) successfully", len(args.Paths)),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) move(ctx context.Context, req *mcp.CallToolRequest, args MoveInput) (*mcp.CallToolResult, any, error) {
	// Resolve destination
	destID, err := s.client.ResolvePath(ctx, args.DestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve destination path '%s': %w", args.DestPath, err)
	}

	// Resolve source paths
	fids := make([]string, 0, len(args.SourcePaths))
	for _, path := range args.SourcePaths {
		fileID, _, _, err := s.client.ResolvePathToFileOrFolder(ctx, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve source path '%s': %w", path, err)
		}
		fids = append(fids, fileID)
	}

	err = s.client.Move(ctx, fids, destID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to move: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Moved %d item(s) to '%s' successfully", len(args.SourcePaths), args.DestPath),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) copy(ctx context.Context, req *mcp.CallToolRequest, args CopyInput) (*mcp.CallToolResult, any, error) {
	// Resolve destination
	destID, err := s.client.ResolvePath(ctx, args.DestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve destination path '%s': %w", args.DestPath, err)
	}

	// Resolve source paths
	fids := make([]string, 0, len(args.SourcePaths))
	for _, path := range args.SourcePaths {
		fileID, _, _, err := s.client.ResolvePathToFileOrFolder(ctx, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve source path '%s': %w", path, err)
		}
		fids = append(fids, fileID)
	}

	err = s.client.Copy(ctx, fids, destID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to copy: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Copied %d item(s) to '%s' successfully", len(args.SourcePaths), args.DestPath),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) getInfo(ctx context.Context, req *mcp.CallToolRequest, args GetInfoInput) (*mcp.CallToolResult, any, error) {
	_, _, fileObj, err := s.client.ResolvePathToFileOrFolder(ctx, args.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", args.Path, err)
	}

	if fileObj == nil {
		return nil, nil, fmt.Errorf("file object not found for path '%s'", args.Path)
	}

	result := map[string]interface{}{
		"name":       fileObj.Name,
		"size":       fileObj.Size,
		"is_folder":  fileObj.IsFolder,
		"updated_at": fileObj.UpdatedAt.Format("2006-01-02 15:04:05"),
		"created_at": fileObj.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) downloadFile(ctx context.Context, req *mcp.CallToolRequest, args DownloadFileInput) (*mcp.CallToolResult, any, error) {
	fileID, isFolder, fileObj, err := s.client.ResolvePathToFileOrFolder(ctx, args.SourcePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", args.SourcePath, err)
	}

	if isFolder {
		return nil, nil, fmt.Errorf("cannot download a folder: '%s'", args.SourcePath)
	}

	taskID := s.downloadMgr.StartDownload(ctx, fileID, args.SourcePath, args.LocalPath)

	result := map[string]interface{}{
		"success":    true,
		"task_id":    taskID,
		"file_name":  fileObj.Name,
		"file_size":  fileObj.Size,
		"local_path": args.LocalPath,
		"message":    fmt.Sprintf("Download task started. Use get_download_status with task_id '%s' to check progress", taskID),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) getDownloadStatus(ctx context.Context, req *mcp.CallToolRequest, args GetDownloadStatusInput) (*mcp.CallToolResult, any, error) {
	task := s.downloadMgr.GetTask(args.TaskID)
	if task == nil {
		return nil, nil, fmt.Errorf("task not found: %s", args.TaskID)
	}

	data, _ := json.MarshalIndent(task.ToMap(), "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) cancelDownload(ctx context.Context, req *mcp.CallToolRequest, args CancelDownloadInput) (*mcp.CallToolResult, any, error) {
	err := s.downloadMgr.CancelTask(args.TaskID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to cancel task: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Download task '%s' canceled", args.TaskID),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) listDownloads(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
	tasks := s.downloadMgr.ListTasks()

	results := make([]map[string]interface{}, len(tasks))
	for i, task := range tasks {
		results[i] = task.ToMap()
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) uploadFile(ctx context.Context, req *mcp.CallToolRequest, args UploadFileInput) (*mcp.CallToolResult, any, error) {
	parentID, err := s.client.ResolvePath(ctx, args.DestPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve destination path '%s': %w", args.DestPath, err)
	}

	err = s.client.UploadFile(ctx, parentID, args.LocalPath, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to upload file: %w", err)
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("File '%s' uploaded to '%s' successfully", args.LocalPath, args.DestPath),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) regexRename(ctx context.Context, req *mcp.CallToolRequest, args RegexRenameInput) (*mcp.CallToolResult, any, error) {
	folderID, err := s.client.ResolvePath(ctx, args.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", args.Path, err)
	}

	renamed, err := s.client.RegexRename(ctx, folderID, args.Pattern, args.Replacement)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to regex rename: %w", err)
	}

	// Convert to simple names for output
	names := make([]string, len(renamed))
	for i, f := range renamed {
		names[i] = f.Name
	}

	result := map[string]interface{}{
		"success":       true,
		"renamed_count": len(renamed),
		"renamed_files": names,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) search(ctx context.Context, req *mcp.CallToolRequest, args SearchInput) (*mcp.CallToolResult, any, error) {
	folderID, err := s.client.ResolvePath(ctx, args.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve path '%s': %w", args.Path, err)
	}

	results, err := s.client.Search(ctx, folderID, args.Keyword)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search: %w", err)
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) getShareFileList(ctx context.Context, req *mcp.CallToolRequest, args GetShareFileListInput) (*mcp.CallToolResult, any, error) {
	// Extract pwd_id and passcode from share URL
	pwdID, passcode := s.client.ExtractShareURL(args.ShareURL)
	if pwdID == "" {
		return nil, nil, fmt.Errorf("invalid share URL: cannot extract share ID from '%s'", args.ShareURL)
	}

	// Get stoken
	stoken, err := s.client.GetStoken(ctx, pwdID, passcode)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get share token: %w", err)
	}

	// Get file list from share root
	files, err := s.client.GetShareDetail(ctx, pwdID, stoken, "0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get share file list: %w", err)
	}

	// Convert to output format
	result := make([]map[string]interface{}, len(files))
	for i, f := range files {
		result[i] = map[string]interface{}{
			"id":         f.ID,
			"name":       f.Name,
			"size":       f.Size,
			"is_folder":  f.IsFolder,
			"category":   f.Category,
			"updated_at": f.UpdatedAt.Format("2006-01-02 15:04:05"),
			"created_at": f.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) getShareDetail(ctx context.Context, req *mcp.CallToolRequest, args GetShareDetailInput) (*mcp.CallToolResult, any, error) {
	// Extract pwd_id and passcode from share URL
	pwdID, passcode := s.client.ExtractShareURL(args.ShareURL)
	if pwdID == "" {
		return nil, nil, fmt.Errorf("invalid share URL: cannot extract share ID from '%s'", args.ShareURL)
	}

	// Get stoken
	stoken, err := s.client.GetStoken(ctx, pwdID, passcode)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get share token: %w", err)
	}

	// Default to root if folder_id is empty
	folderID := args.FolderID
	if folderID == "" {
		folderID = "0"
	}

	// Get file list from the specified folder
	files, err := s.client.GetShareDetail(ctx, pwdID, stoken, folderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get share detail for folder '%s': %w", folderID, err)
	}

	// Convert to output format
	result := make([]map[string]interface{}, len(files))
	for i, f := range files {
		result[i] = map[string]interface{}{
			"id":         f.ID,
			"name":       f.Name,
			"size":       f.Size,
			"is_folder":  f.IsFolder,
			"category":   f.Category,
			"updated_at": f.UpdatedAt.Format("2006-01-02 15:04:05"),
			"created_at": f.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func (s *Server) saveFromShare(ctx context.Context, req *mcp.CallToolRequest, args SaveFromShareInput) (*mcp.CallToolResult, any, error) {
	// Extract pwd_id and passcode from share URL
	pwdID, passcode := s.client.ExtractShareURL(args.ShareURL)
	if pwdID == "" {
		return nil, nil, fmt.Errorf("invalid share URL: cannot extract share ID from '%s'", args.ShareURL)
	}

	// Get stoken
	stoken, err := s.client.GetStoken(ctx, pwdID, passcode)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get share token: %w", err)
	}

	// Determine the folder to get files from
	folderID := args.FolderID
	if folderID == "" {
		folderID = "0"
	}

	// Get file list from the specified folder
	allFiles, err := s.client.GetShareDetail(ctx, pwdID, stoken, folderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get share file list from folder '%s': %w", folderID, err)
	}

	// Determine which files to save
	var filesToSave []map[string]interface{}
	if len(args.FileIDs) > 0 {
		// Save specific files
		for _, f := range allFiles {
			for _, id := range args.FileIDs {
				if f.ID == id {
					filesToSave = append(filesToSave, map[string]interface{}{
						"fid":       f.ID,
						"name":      f.Name,
						"is_folder": f.IsFolder,
					})
					break
				}
			}
		}
		if len(filesToSave) == 0 {
			return nil, nil, fmt.Errorf("no matching files found for the specified IDs in folder '%s'", folderID)
		}
	} else {
		// Save all files from the folder
		for _, f := range allFiles {
			filesToSave = append(filesToSave, map[string]interface{}{
				"fid":       f.ID,
				"name":      f.Name,
				"is_folder": f.IsFolder,
			})
		}
	}

	// Get raw share detail to get fid_token_list from the same folder
	rawFiles, err := s.client.GetShareDetailRaw(ctx, pwdID, stoken, folderID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get raw share detail: %w", err)
	}

	// Map fid to fid_token
	fidTokenMap := make(map[string]string)
	for _, f := range rawFiles {
		fidTokenMap[f.Fid] = f.ShareFidToken
	}

	// Build fid_list and fid_token_list
	fidList := make([]string, len(filesToSave))
	fidTokenList := make([]string, len(filesToSave))
	fileNames := make([]string, len(filesToSave))

	for i, f := range filesToSave {
		fid := f["fid"].(string)
		fidList[i] = fid
		fidTokenList[i] = fidTokenMap[fid]
		fileNames[i] = f["name"].(string)
	}

	// Resolve destination path
	destFid, err := s.client.ResolvePath(ctx, args.DestPath)
	if err != nil {
		// Try to create the folder if it doesn't exist
		parentID, folderName, err2 := s.client.ResolveParentPathAndName(ctx, args.DestPath)
		if err2 != nil {
			return nil, nil, fmt.Errorf("failed to resolve destination path '%s': %w", args.DestPath, err)
		}
		destFid, err = s.client.CreateFolder(ctx, parentID, folderName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create destination folder: %w", err)
		}
	}

	// Save files from share
	taskID, err := s.client.SaveFromShare(ctx, pwdID, stoken, fidList, fidTokenList, destFid)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save files from share: %w", err)
	}

	// Wait for task to complete
	savedFids, err := s.client.WaitForTask(ctx, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("save task failed: %w", err)
	}

	result := map[string]interface{}{
		"success":     true,
		"task_id":     taskID,
		"saved_count": len(savedFids),
		"saved_files": fileNames,
		"dest_path":   args.DestPath,
		"folder_id":   folderID,
		"message":     fmt.Sprintf("Saved %d file(s) to '%s'", len(savedFids), args.DestPath),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}
