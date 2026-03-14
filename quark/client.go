package quark

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	APIEndpoint = "https://drive.quark.cn/1/clouddrive"
	UserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) quark-cloud-drive/2.5.20 Chrome/100.0.4896.160 Electron/18.3.5.4-b478491100 Safari/537.36 Channel/pckk_other_ch"
	Referer      = "https://pan.quark.cn"
	PR           = "ucpro"
)

// Client is the Quark API client
type Client struct {
	cookie    string
	client    *resty.Client
	orderBy   string
	orderDir  string
}

// NewClient creates a new Quark API client
func NewClient(cookie string) *Client {
	client := resty.New()
	client.SetTimeout(60 * time.Second)

	return &Client{
		cookie:   cookie,
		client:   client,
		orderBy:  "none",
		orderDir: "asc",
	}
}

// SetOrder sets the ordering for file listing
func (c *Client) SetOrder(orderBy, orderDir string) {
	c.orderBy = orderBy
	c.orderDir = orderDir
}

// request makes an API request to Quark
func (c *Client) request(ctx context.Context, pathname string, method string, callback func(*resty.Request), resp interface{}) ([]byte, error) {
	u := APIEndpoint + pathname
	req := c.client.R().SetContext(ctx)
	req.SetHeaders(map[string]string{
		"Cookie":     c.cookie,
		"Accept":     "application/json, text/plain, */*",
		"Referer":    Referer,
		"User-Agent": UserAgent,
	})
	req.SetQueryParam("pr", PR)
	req.SetQueryParam("fr", "pc")

	if callback != nil {
		callback(req)
	}
	if resp != nil {
		req.SetResult(resp)
	}

	var e Resp
	req.SetError(&e)

	res, err := req.Execute(method, u)
	if err != nil {
		return nil, err
	}

	if e.Status >= 400 || e.Code != 0 {
		return nil, fmt.Errorf("API error: %s (status: %d, code: %d)", e.Message, e.Status, e.Code)
	}

	return res.Body(), nil
}

// resolvePath resolves a path like "/folder1/folder2" to a folder ID
// Returns the folder ID, or error if not found
// If path is "/" or "", returns "0" (root)
func (c *Client) resolvePath(ctx context.Context, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "0", nil
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")
	currentID := "0"

	for _, part := range parts {
		if part == "" {
			continue
		}

		files, err := c.ListFiles(ctx, currentID)
		if err != nil {
			return "", fmt.Errorf("failed to list folder: %w", err)
		}

		found := false
		for _, f := range files {
			if f.Name == part {
				if !f.IsFolder {
					return "", fmt.Errorf("'%s' is a file, not a folder", part)
				}
				currentID = f.ID
				found = true
				break
			}
		}

		if !found {
			return "", fmt.Errorf("folder '%s' not found", part)
		}
	}

	return currentID, nil
}

// resolvePathToFileOrFolder resolves a path to a file or folder ID
// Returns the ID, whether it's a folder, and error
func (c *Client) resolvePathToFileOrFolder(ctx context.Context, path string) (string, bool, *FileObj, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "0", true, &FileObj{ID: "0", Name: "", IsFolder: true}, nil
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")
	currentID := "0"

	for i, part := range parts {
		if part == "" {
			continue
		}

		files, err := c.ListFiles(ctx, currentID)
		if err != nil {
			return "", false, nil, fmt.Errorf("failed to list folder: %w", err)
		}

		found := false
		for _, f := range files {
			if f.Name == part {
				currentID = f.ID
				found = true
				// If this is the last part, return the result
				if i == len(parts)-1 {
					return f.ID, f.IsFolder, f, nil
				}
				// If not the last part but it's a file, error
				if !f.IsFolder {
					return "", false, nil, fmt.Errorf("'%s' is a file, not a folder", part)
				}
				break
			}
		}

		if !found {
			return "", false, nil, fmt.Errorf("'%s' not found", part)
		}
	}

	return currentID, true, nil, nil
}

// resolveParentPathAndName resolves a path to parent folder ID and the last name component
// e.g., "/folder1/folder2/newname" -> ("folder2_id", "newname")
func (c *Client) resolveParentPathAndName(ctx context.Context, path string) (string, string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", "", fmt.Errorf("path cannot be empty")
	}

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", "", fmt.Errorf("invalid path")
	}

	// The last part is the name
	name := parts[len(parts)-1]
	if name == "" {
		return "", "", fmt.Errorf("name cannot be empty")
	}

	// Resolve parent path
	if len(parts) == 1 {
		return "0", name, nil
	}

	parentPath := strings.Join(parts[:len(parts)-1], "/")
	parentID, err := c.resolvePath(ctx, "/"+parentPath)
	if err != nil {
		return "", "", err
	}

	return parentID, name, nil
}

// ResolvePath is the exported version of resolvePath
func (c *Client) ResolvePath(ctx context.Context, path string) (string, error) {
	return c.resolvePath(ctx, path)
}

// ResolveParentPathAndName is the exported version of resolveParentPathAndName
func (c *Client) ResolveParentPathAndName(ctx context.Context, path string) (string, string, error) {
	return c.resolveParentPathAndName(ctx, path)
}

// ResolvePathToFileOrFolder is the exported version of resolvePathToFileOrFolder
func (c *Client) ResolvePathToFileOrFolder(ctx context.Context, path string) (string, bool, *FileObj, error) {
	return c.resolvePathToFileOrFolder(ctx, path)
}

// Search searches for files/folders matching keyword in a directory recursively
func (c *Client) Search(ctx context.Context, parentFid, keyword string) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0)
	keyword = strings.ToLower(keyword)

	var searchRecursive func(folderID, currentPath string) error
	searchRecursive = func(folderID, currentPath string) error {
		files, err := c.ListFiles(ctx, folderID)
		if err != nil {
			return err
		}

		for _, f := range files {
			fullPath := currentPath + "/" + f.Name
			if strings.Contains(strings.ToLower(f.Name), keyword) {
				results = append(results, map[string]interface{}{
					"path":       fullPath,
					"name":       f.Name,
					"size":       f.Size,
					"is_folder":  f.IsFolder,
					"updated_at": f.UpdatedAt.Format("2006-01-02 15:04:05"),
				})
			}

			// Search in subdirectories
			if f.IsFolder {
				if err := searchRecursive(f.ID, fullPath); err != nil {
					continue
				}
			}
		}
		return nil
	}

	if err := searchRecursive(parentFid, ""); err != nil {
		return nil, err
	}

	return results, nil
}

// ListFiles lists files and folders in a directory
func (c *Client) ListFiles(ctx context.Context, parentFid string) ([]*FileObj, error) {
	files := make([]File, 0)
	page := 1
	size := 100

	query := map[string]string{
		"pdir_fid":     parentFid,
		"_size":        strconv.Itoa(size),
		"_fetch_total": "1",
	}

	if c.orderBy != "none" {
		query["_sort"] = "file_type:asc," + c.orderBy + ":" + c.orderDir
	}

	for {
		query["_page"] = strconv.Itoa(page)
		var resp SortResp
		_, err := c.request(ctx, "/file/sort", http.MethodGet, func(req *resty.Request) {
			req.SetQueryParams(query)
		}, &resp)
		if err != nil {
			return nil, err
		}

		files = append(files, resp.Data.List...)

		if page*size >= resp.Metadata.Total {
			break
		}
		page++
	}

	result := make([]*FileObj, len(files))
	for i, f := range files {
		result[i] = f.ToFileObj()
	}
	return result, nil
}

// CreateFolder creates a new folder
func (c *Client) CreateFolder(ctx context.Context, parentFid, folderName string) (string, error) {
	data := map[string]interface{}{
		"dir_init_lock": false,
		"dir_path":      "",
		"file_name":     folderName,
		"pdir_fid":      parentFid,
	}

	var resp CreateFolderResp
	_, err := c.request(ctx, "/file", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, &resp)
	if err != nil {
		return "", err
	}

	return resp.Data.Fid, nil
}

// Rename renames a file or folder
func (c *Client) Rename(ctx context.Context, fid, newName string) error {
	data := map[string]interface{}{
		"fid":        fid,
		"file_name":   newName,
	}

	_, err := c.request(ctx, "/file/rename", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

// Delete deletes files or folders
func (c *Client) Delete(ctx context.Context, fids []string) error {
	data := map[string]interface{}{
		"action_type":  1,
		"exclude_fids": []string{},
		"filelist":     fids,
	}

	_, err := c.request(ctx, "/file/delete", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

// Move moves files or folders to a destination
func (c *Client) Move(ctx context.Context, fids []string, destFid string) error {
	data := map[string]interface{}{
		"action_type":  1,
		"exclude_fids": []string{},
		"filelist":     fids,
		"to_pdir_fid":  destFid,
	}

	_, err := c.request(ctx, "/file/move", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

// Copy copies files or folders to a destination
func (c *Client) Copy(ctx context.Context, fids []string, destFid string) error {
	data := map[string]interface{}{
		"action_type":  1,
		"exclude_fids": []string{},
		"filelist":     fids,
		"to_pdir_fid":  destFid,
	}

	_, err := c.request(ctx, "/file/copy", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

// GetInfo gets information about a file or folder by searching from root
// Note: This iterates through directories to find the file, which may be slow
func (c *Client) GetInfo(ctx context.Context, fid string) (*FileObj, error) {
	// Try to find the file by listing all files in root and subdirectories
	// This is a workaround since there's no direct API to get file info by fid
	var findFile func(parentFid string) (*FileObj, error)
	findFile = func(parentFid string) (*FileObj, error) {
		files, err := c.ListFiles(ctx, parentFid)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if f.ID == fid {
				return f, nil
			}
			// Search in subdirectories
			if f.IsFolder {
				found, err := findFile(f.ID)
				if err != nil {
					continue
				}
				if found != nil {
					return found, nil
				}
			}
		}
		return nil, nil
	}

	// Only search in root directory for performance
	files, err := c.ListFiles(ctx, "0")
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.ID == fid {
			return f, nil
		}
	}

	// Also search one level deep
	for _, f := range files {
		if f.IsFolder {
			subFiles, err := c.ListFiles(ctx, f.ID)
			if err != nil {
				continue
			}
			for _, sf := range subFiles {
				if sf.ID == fid {
					return sf, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("file with ID %s not found", fid)
}

// GetDownloadUrl gets the download URL for a file
func (c *Client) GetDownloadUrl(ctx context.Context, fid string) (string, error) {
	data := map[string]interface{}{
		"fids": []string{fid},
	}

	var resp DownResp
	_, err := c.request(ctx, "/file/download", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, &resp)
	if err != nil {
		return "", err
	}

	if len(resp.Data) == 0 {
		return "", errors.New("no download URL returned")
	}

	return resp.Data[0].DownloadUrl, nil
}

// UploadFile uploads a file to a folder
func (c *Client) UploadFile(ctx context.Context, parentFid, filePath string, progressCallback func(int64, int64)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fileSize := stat.Size()
	fileName := filepath.Base(filePath)

	// Calculate MD5 and SHA1
	md5Hash := md5.New()
	sha1Hash := md5.New() // Note: Quark uses md5 for both in their implementation
	if _, err := io.Copy(io.MultiWriter(md5Hash, sha1Hash), file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	md5Str := hex.EncodeToString(md5Hash.Sum(nil))
	sha1Str := hex.EncodeToString(sha1Hash.Sum(nil))

	// Pre-upload
	now := time.Now()
	preData := map[string]interface{}{
		"ccp_hash_update": true,
		"dir_name":        "",
		"file_name":       fileName,
		"format_type":     "application/octet-stream",
		"l_created_at":    now.UnixMilli(),
		"l_updated_at":    now.UnixMilli(),
		"pdir_fid":        parentFid,
		"size":            fileSize,
	}

	var preResp UpPreResp
	_, err = c.request(ctx, "/file/upload/pre", http.MethodPost, func(req *resty.Request) {
		req.SetBody(preData)
	}, &preResp)
	if err != nil {
		return fmt.Errorf("upload pre-check failed: %w", err)
	}

	// Check hash for quick upload
	hashData := map[string]interface{}{
		"md5":     md5Str,
		"sha1":    sha1Str,
		"task_id": preResp.Data.TaskId,
	}

	var hashResp HashResp
	_, err = c.request(ctx, "/file/update/hash", http.MethodPost, func(req *resty.Request) {
		req.SetBody(hashData)
	}, &hashResp)
	if err != nil {
		return fmt.Errorf("hash check failed: %w", err)
	}

	if hashResp.Data.Finish {
		// File already exists, quick upload complete
		return c.uploadFinish(ctx, preResp.Data.TaskId, preResp.Data.ObjKey)
	}

	// Upload parts
	partSize := int64(preResp.Metadata.PartSize)
	if partSize == 0 {
		partSize = 10 * 1024 * 1024 // Default 10MB
	}

	partNumber := 1
	var md5s []string
	var uploaded int64 = 0

	for uploaded < fileSize {
		part := make([]byte, partSize)
		n, err := file.Read(part)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read file part: %w", err)
		}
		if n == 0 {
			break
		}
		part = part[:n]

		etag, err := c.uploadPart(ctx, preResp, partNumber, part)
		if err != nil {
			return fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}

		md5s = append(md5s, etag)
		uploaded += int64(n)
		partNumber++

		if progressCallback != nil {
			progressCallback(uploaded, fileSize)
		}
	}

	// Commit upload
	if err := c.uploadCommit(ctx, preResp, md5s); err != nil {
		return fmt.Errorf("failed to commit upload: %w", err)
	}

	return c.uploadFinish(ctx, preResp.Data.TaskId, preResp.Data.ObjKey)
}

func (c *Client) uploadPart(ctx context.Context, pre UpPreResp, partNumber int, part []byte) (string, error) {
	timeStr := time.Now().UTC().Format(http.TimeFormat)
	mimeType := "application/octet-stream"

	data := map[string]interface{}{
		"auth_info": pre.Data.AuthInfo,
		"auth_meta": fmt.Sprintf(`PUT

%s
%s
x-oss-date:%s
x-oss-user-agent:aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit
/%s/%s?partNumber=%d&uploadId=%s`,
			mimeType, timeStr, timeStr, pre.Data.Bucket, pre.Data.ObjKey, partNumber, pre.Data.UploadId),
		"task_id": pre.Data.TaskId,
	}

	var resp UpAuthResp
	_, err := c.request(ctx, "/file/upload/auth", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, &resp)
	if err != nil {
		return "", err
	}

	u := fmt.Sprintf("https://%s.%s/%s", pre.Data.Bucket, pre.Data.UploadUrl[7:], pre.Data.ObjKey)
	res, err := c.client.R().SetContext(ctx).
		SetHeaders(map[string]string{
			"Authorization":    resp.Data.AuthKey,
			"Content-Type":     mimeType,
			"Referer":          Referer,
			"x-oss-date":       timeStr,
			"x-oss-user-agent": "aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit",
		}).
		SetQueryParams(map[string]string{
			"partNumber": strconv.Itoa(partNumber),
			"uploadId":   pre.Data.UploadId,
		}).
		SetBody(part).
		Put(u)

	if err != nil {
		return "", err
	}
	if res.StatusCode() != 200 {
		return "", fmt.Errorf("upload part failed with status: %d", res.StatusCode())
	}

	return res.Header().Get("Etag"), nil
}

func (c *Client) uploadCommit(ctx context.Context, pre UpPreResp, md5s []string) error {
	timeStr := time.Now().UTC().Format(http.TimeFormat)

	var bodyBuilder strings.Builder
	bodyBuilder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<CompleteMultipartUpload>
`)
	for i, m := range md5s {
		bodyBuilder.WriteString(fmt.Sprintf(`<Part>
<PartNumber>%d</PartNumber>
<ETag>%s</ETag>
</Part>
`, i+1, m))
	}
	bodyBuilder.WriteString("</CompleteMultipartUpload>")
	body := bodyBuilder.String()

	m := md5.New()
	m.Write([]byte(body))
	contentMd5 := base64.StdEncoding.EncodeToString(m.Sum(nil))

	callbackBytes, _ := jsonMarshal(pre.Data.Callback)
	callbackBase64 := base64.StdEncoding.EncodeToString(callbackBytes)

	data := map[string]interface{}{
		"auth_info": pre.Data.AuthInfo,
		"auth_meta": fmt.Sprintf(`POST
%s
application/xml
%s
x-oss-callback:%s
x-oss-date:%s
x-oss-user-agent:aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit
/%s/%s?uploadId=%s`,
			contentMd5, timeStr, callbackBase64, timeStr,
			pre.Data.Bucket, pre.Data.ObjKey, pre.Data.UploadId),
		"task_id": pre.Data.TaskId,
	}

	var resp UpAuthResp
	_, err := c.request(ctx, "/file/upload/auth", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, &resp)
	if err != nil {
		return err
	}

	u := fmt.Sprintf("https://%s.%s/%s", pre.Data.Bucket, pre.Data.UploadUrl[7:], pre.Data.ObjKey)
	res, err := c.client.R().
		SetHeaders(map[string]string{
			"Authorization":    resp.Data.AuthKey,
			"Content-MD5":      contentMd5,
			"Content-Type":     "application/xml",
			"Referer":          Referer,
			"x-oss-callback":   callbackBase64,
			"x-oss-date":       timeStr,
			"x-oss-user-agent": "aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit",
		}).
		SetQueryParams(map[string]string{
			"uploadId": pre.Data.UploadId,
		}).
		SetBody(body).
		Post(u)

	if err != nil {
		return err
	}
	if res.StatusCode() != 200 {
		return fmt.Errorf("commit failed with status: %d", res.StatusCode())
	}

	return nil
}

func (c *Client) uploadFinish(ctx context.Context, taskId, objKey string) error {
	data := map[string]interface{}{
		"obj_key": objKey,
		"task_id": taskId,
	}

	_, err := c.request(ctx, "/file/upload/finish", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)

	time.Sleep(time.Second)
	return err
}

// RegexRename renames files matching a regex pattern
func (c *Client) RegexRename(ctx context.Context, parentFid, pattern, replacement string) ([]*FileObj, error) {
	files, err := c.ListFiles(ctx, parentFid)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	var renamed []*FileObj
	for _, file := range files {
		newName := re.ReplaceAllString(file.Name, replacement)
		if newName != file.Name && newName != "" {
			if err := c.Rename(ctx, file.ID, newName); err != nil {
				return renamed, fmt.Errorf("failed to rename %s to %s: %w", file.Name, newName, err)
			}
			file.Name = newName
			renamed = append(renamed, file)
		}
	}

	return renamed, nil
}

// UploadFromReader uploads a file from an io.Reader
func (c *Client) UploadFromReader(ctx context.Context, parentFid, fileName string, reader io.Reader, size int64, mimeType string, progressCallback func(int64, int64)) error {
	// Read all data into memory for hash calculation
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	md5Hash := md5.New()
	sha1Hash := md5.New()
	if _, err := io.Copy(io.MultiWriter(md5Hash, sha1Hash), bytes.NewReader(data)); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	md5Str := hex.EncodeToString(md5Hash.Sum(nil))
	sha1Str := hex.EncodeToString(sha1Hash.Sum(nil))

	// Pre-upload
	now := time.Now()
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	preData := map[string]interface{}{
		"ccp_hash_update": true,
		"dir_name":        "",
		"file_name":       fileName,
		"format_type":     mimeType,
		"l_created_at":    now.UnixMilli(),
		"l_updated_at":    now.UnixMilli(),
		"pdir_fid":        parentFid,
		"size":            size,
	}

	var preResp UpPreResp
	_, err = c.request(ctx, "/file/upload/pre", http.MethodPost, func(req *resty.Request) {
		req.SetBody(preData)
	}, &preResp)
	if err != nil {
		return fmt.Errorf("upload pre-check failed: %w", err)
	}

	// Check hash
	hashData := map[string]interface{}{
		"md5":     md5Str,
		"sha1":    sha1Str,
		"task_id": preResp.Data.TaskId,
	}

	var hashResp HashResp
	_, err = c.request(ctx, "/file/update/hash", http.MethodPost, func(req *resty.Request) {
		req.SetBody(hashData)
	}, &hashResp)
	if err != nil {
		return fmt.Errorf("hash check failed: %w", err)
	}

	if hashResp.Data.Finish {
		return c.uploadFinish(ctx, preResp.Data.TaskId, preResp.Data.ObjKey)
	}

	// Upload parts
	partSize := int64(preResp.Metadata.PartSize)
	if partSize == 0 {
		partSize = 10 * 1024 * 1024
	}

	partNumber := 1
	var md5s []string
	var uploaded int64 = 0

	for uploaded < size {
		end := uploaded + partSize
		if end > size {
			end = size
		}
		part := data[uploaded:end]

		etag, err := c.uploadPart(ctx, preResp, partNumber, part)
		if err != nil {
			return fmt.Errorf("failed to upload part %d: %w", partNumber, err)
		}

		md5s = append(md5s, etag)
		uploaded = end
		partNumber++

		if progressCallback != nil {
			progressCallback(uploaded, size)
		}
	}

	if err := c.uploadCommit(ctx, preResp, md5s); err != nil {
		return fmt.Errorf("failed to commit upload: %w", err)
	}

	return c.uploadFinish(ctx, preResp.Data.TaskId, preResp.Data.ObjKey)
}

func jsonMarshal(v interface{}) ([]byte, error) {
	// Simple JSON marshal for callback struct
	var buf bytes.Buffer
	buf.WriteByte('{')
	switch t := v.(type) {
	case struct {
		CallbackUrl  string `json:"callbackUrl"`
		CallbackBody string `json:"callbackBody"`
	}:
		buf.WriteString(`"callbackUrl":"`)
		buf.WriteString(t.CallbackUrl)
		buf.WriteString(`","callbackBody":"`)
		buf.WriteString(t.CallbackBody)
		buf.WriteString(`"`)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}