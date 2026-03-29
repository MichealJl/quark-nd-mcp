package quark

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DownloadStatus represents the status of a download task
type DownloadStatus string

const (
	DownloadStatusPending   DownloadStatus = "pending"
	DownloadStatusRunning   DownloadStatus = "running"
	DownloadStatusCompleted DownloadStatus = "completed"
	DownloadStatusFailed    DownloadStatus = "failed"
	DownloadStatusCanceled  DownloadStatus = "canceled"
)

// DownloadTask represents a download task
type DownloadTask struct {
	ID           string
	SourcePath   string
	LocalPath    string
	Status       DownloadStatus
	Progress     float64 // 0-100
	Downloaded   int64
	TotalSize    int64
	Error        string
	StartTime    time.Time
	EndTime      time.Time
	cancelFunc   context.CancelFunc
}

// DownloadManager manages download tasks
type DownloadManager struct {
	client *Client
	tasks  map[string]*DownloadTask
	mu     sync.RWMutex
}

// NewDownloadManager creates a new download manager
func NewDownloadManager(client *Client) *DownloadManager {
	return &DownloadManager{
		client: client,
		tasks:  make(map[string]*DownloadTask),
	}
}

// StartDownload starts a new download task and returns the task ID
func (dm *DownloadManager) StartDownload(ctx context.Context, fid, sourcePath, localPath string) string {
	taskID := fmt.Sprintf("dl_%d", time.Now().UnixNano())

	task := &DownloadTask{
		ID:         taskID,
		SourcePath: sourcePath,
		LocalPath:  localPath,
		Status:     DownloadStatusPending,
		StartTime:  time.Now(),
	}

	dm.mu.Lock()
	dm.tasks[taskID] = task
	dm.mu.Unlock()

	// Create a cancellable context for this task
	taskCtx, cancel := context.WithCancel(context.Background())
	task.cancelFunc = cancel

	// Start download in background
	go dm.runDownload(taskCtx, task, fid)

	return taskID
}

// runDownload executes the download task
func (dm *DownloadManager) runDownload(ctx context.Context, task *DownloadTask, fid string) {
	dm.mu.Lock()
	task.Status = DownloadStatusRunning
	dm.mu.Unlock()

	progressCallback := func(downloaded, total int64) {
		dm.mu.Lock()
		task.Downloaded = downloaded
		task.TotalSize = total
		if total > 0 {
			task.Progress = float64(downloaded) / float64(total) * 100
		}
		dm.mu.Unlock()
	}

	err := dm.client.DownloadFile(ctx, fid, task.LocalPath, progressCallback)

	dm.mu.Lock()
	defer dm.mu.Unlock()

	if err != nil {
		if ctx.Err() == context.Canceled {
			task.Status = DownloadStatusCanceled
		} else {
			task.Status = DownloadStatusFailed
			task.Error = err.Error()
		}
	} else {
		task.Status = DownloadStatusCompleted
		task.Progress = 100
	}
	task.EndTime = time.Now()
}

// GetTask returns the task status by ID
func (dm *DownloadManager) GetTask(taskID string) *DownloadTask {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.tasks[taskID]
}

// ListTasks returns all tasks
func (dm *DownloadManager) ListTasks() []*DownloadTask {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	tasks := make([]*DownloadTask, 0, len(dm.tasks))
	for _, task := range dm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// CancelTask cancels a running task
func (dm *DownloadManager) CancelTask(taskID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	task, ok := dm.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status == DownloadStatusRunning && task.cancelFunc != nil {
		task.cancelFunc()
		task.Status = DownloadStatusCanceled
		return nil
	}

	return fmt.Errorf("task is not running: %s", task.Status)
}

// ClearCompletedTasks removes completed/failed/canceled tasks
func (dm *DownloadManager) ClearCompletedTasks() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	for id, task := range dm.tasks {
		if task.Status == DownloadStatusCompleted ||
			task.Status == DownloadStatusFailed ||
			task.Status == DownloadStatusCanceled {
			delete(dm.tasks, id)
		}
	}
}

// ToMap converts task to a map for JSON output
func (t *DownloadTask) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":          t.ID,
		"source_path": t.SourcePath,
		"local_path":  t.LocalPath,
		"status":      t.Status,
		"progress":    fmt.Sprintf("%.2f%%", t.Progress),
		"downloaded":  t.Downloaded,
		"total_size":  t.TotalSize,
		"error":       t.Error,
		"start_time":  t.StartTime.Format("2006-01-02 15:04:05"),
		"end_time":    t.EndTime.Format("2006-01-02 15:04:05"),
	}
}