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
	client         *quark.Client
	server         *mcp.Server
	downloadMgr    *quark.DownloadManager
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
		Description: "List files and folders in a directory by path. Use '/' or '' for root directory.",
	}, s.listFiles)

	// Tool: Create folder
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "create_folder",
		Description: "Create a new folder at the specified path. Example: '/parent_folder/new_folder' or 'new_folder' for root.",
	}, s.createFolder)

	// Tool: Rename
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "rename",
		Description: "Rename a file or folder by path. Example: '/folder/old_name' -> '/folder/new_name'",
	}, s.rename)

	// Tool: Delete
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "delete",
		Description: "Delete files or folders by path. Example: '/folder/file.txt' or ['/folder1', '/folder2/file.txt']",
	}, s.delete)

	// Tool: Move
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "move",
		Description: "Move files or folders to a destination directory. Use paths like '/source/file.txt' and '/destination/'. Use '/' for root.",
	}, s.move)

	// Tool: Copy
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "copy",
		Description: "Copy files or folders to a destination directory. Use paths like '/source/file.txt' and '/destination/'. Use '/' for root.",
	}, s.copy)

	// Tool: Get info
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_info",
		Description: "Get detailed information about a file or folder by path. Example: '/folder/file.txt'",
	}, s.getInfo)

	// Tool: Download file (async)
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "download_file",
		Description: "Start an async download task. Returns task ID. Use get_download_status to check progress. Example: '/folder/file.txt' to '/Users/name/Downloads/file.txt'",
	}, s.downloadFile)

	// Tool: Get download status
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_download_status",
		Description: "Get the status of a download task by ID. Returns progress, status (pending/running/completed/failed/canceled).",
	}, s.getDownloadStatus)

	// Tool: Cancel download
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "cancel_download",
		Description: "Cancel a running download task by ID.",
	}, s.cancelDownload)

	// Tool: List downloads
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list_downloads",
		Description: "List all download tasks and their status.",
	}, s.listDownloads)

	// Tool: Upload file
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "upload_file",
		Description: "Upload a local file to Quark drive. Specify destination folder path (use '/' for root) and local file path.",
	}, s.uploadFile)

	// Tool: Regex rename
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "regex_rename",
		Description: "Batch rename files in a directory using regular expression pattern. Example: path='/photos', pattern='IMG_(.*)', replacement='Photo_$1'",
	}, s.regexRename)

	// Tool: Search
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "search",
		Description: "Search for files or folders by name in a directory (recursively). Example: path='/folder', keyword='photo'",
	}, s.search)
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
	DestPath   string `json:"dest_path" jsonschema:"Destination folder path. Use '/' for root. Example: '/我的文档'"`
	LocalPath  string `json:"local_path" jsonschema:"Local file path to upload. Example: '/Users/name/Downloads/file.txt'"`
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