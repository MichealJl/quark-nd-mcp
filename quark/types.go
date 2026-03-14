package quark

import "time"

// Resp is the base response from Quark API
type Resp struct {
	Status  int    `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// File represents a file or folder in Quark drive
type File struct {
	Fid       string `json:"fid"`
	FileName  string `json:"file_name"`
	Size      int64  `json:"size"`
	LUpdated  int64  `json:"l_updated_at"`
	File      bool   `json:"file"`
	UpdatedAt int64  `json:"updated_at"`
	CreatedAt int64  `json:"created_at"`
}

// FileInfo represents detailed file information
type FileInfo struct {
	Fid         string `json:"fid"`
	FileName    string `json:"file_name"`
	Size        int64  `json:"size"`
	File        bool   `json:"file"`
	UpdatedAt   int64  `json:"updated_at"`
	CreatedAt   int64  `json:"created_at"`
	FileType    int    `json:"file_type"`
	FormatType  string `json:"format_type"`
	Category    int    `json:"category"`
}

// FileObj is a simplified file object for MCP responses
type FileObj struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	IsFolder  bool      `json:"is_folder"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ToFileObj converts File to FileObj
func (f *File) ToFileObj() *FileObj {
	return &FileObj{
		ID:        f.Fid,
		Name:      f.FileName,
		Size:      f.Size,
		IsFolder:  !f.File,
		UpdatedAt: time.UnixMilli(f.UpdatedAt),
		CreatedAt: time.UnixMilli(f.CreatedAt),
	}
}

// SortResp is the response for file list
type SortResp struct {
	Resp
	Data struct {
		List []File `json:"list"`
	} `json:"data"`
	Metadata struct {
		Total int `json:"_total"`
	} `json:"metadata"`
}

// DownResp is the response for download URL
type DownResp struct {
	Resp
	Data []struct {
		DownloadUrl string `json:"download_url"`
	} `json:"data"`
}

// UpPreResp is the response for upload pre-check
type UpPreResp struct {
	Resp
	Data struct {
		TaskId    string `json:"task_id"`
		Finish    bool   `json:"finish"`
		UploadId  string `json:"upload_id"`
		ObjKey    string `json:"obj_key"`
		UploadUrl string `json:"upload_url"`
		Fid       string `json:"fid"`
		Bucket    string `json:"bucket"`
		Callback  struct {
			CallbackUrl  string `json:"callbackUrl"`
			CallbackBody string `json:"callbackBody"`
		} `json:"callback"`
		FormatType string `json:"format_type"`
		Size       int    `json:"size"`
		AuthInfo   string `json:"auth_info"`
	} `json:"data"`
	Metadata struct {
		PartSize int `json:"part_size"`
	} `json:"metadata"`
}

// HashResp is the response for upload hash check
type HashResp struct {
	Resp
	Data struct {
		Finish     bool   `json:"finish"`
		Fid        string `json:"fid"`
		Thumbnail  string `json:"thumbnail"`
		FormatType string `json:"format_type"`
	} `json:"data"`
}

// UpAuthResp is the response for upload auth
type UpAuthResp struct {
	Resp
	Data struct {
		AuthKey string        `json:"auth_key"`
		Speed   int           `json:"speed"`
		Headers []interface{} `json:"headers"`
	} `json:"data"`
}

// MoveCopyResp is the response for move/copy operations
type MoveCopyResp struct {
	Resp
	Data struct {
		TaskId string `json:"task_id"`
	} `json:"data"`
}

// CreateFolderResp is the response for creating folder
type CreateFolderResp struct {
	Resp
	Data struct {
		Fid string `json:"fid"`
	} `json:"data"`
}

// GetInfoResp is the response for getting file info
type GetInfoResp struct {
	Resp
	Data struct {
		File FileInfo `json:"file"`
	} `json:"data"`
}