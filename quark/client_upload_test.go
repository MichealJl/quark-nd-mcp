package quark

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type uploadTestTransport func(*http.Request) (*http.Response, error)

func (f uploadTestTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestUploadHashPayload(t *testing.T) {
	tests := []struct{ name, content, md5, sha1 string }{
		{"empty", "", "d41d8cd98f00b204e9800998ecf8427e", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"abc", "abc", "900150983cd24fb0d6963f7d28e17f72", "a9993e364706816aba3e25717850c26c9cd0d89d"},
	}
	for _, method := range []string{"file", "reader"} {
		for _, tt := range tests {
			t.Run(method+"/"+tt.name, func(t *testing.T) {
				c := NewClient("test-cookie")
				var hashes map[string]string
				finished := false
				c.client.SetTransport(uploadTestTransport(func(req *http.Request) (*http.Response, error) {
					if req.Header.Get("Cookie") != "test-cookie" {
						return nil, fmt.Errorf("Cookie header was not forwarded unchanged")
					}
					var body string
					switch req.URL.Path {
					case "/1/clouddrive/file/upload/pre":
						body = `{"status":200,"code":0,"data":{"task_id":"test-task","obj_key":"test-object"}}`
					case "/1/clouddrive/file/update/hash":
						if err := json.NewDecoder(req.Body).Decode(&hashes); err != nil {
							return nil, err
						}
						body = `{"status":200,"code":0,"data":{"finish":true}}`
					case "/1/clouddrive/file/upload/finish":
						finished = true
						body = `{"status":200,"code":0}`
					default:
						return nil, fmt.Errorf("unexpected request: %s", req.URL.Path)
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
				}))
				var err error
				if method == "file" {
					path := filepath.Join(t.TempDir(), "test.bin")
					if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
						t.Fatal(err)
					}
					err = c.UploadFile(context.Background(), "0", path, nil)
				} else {
					err = c.UploadFromReader(context.Background(), "0", "test.bin", strings.NewReader(tt.content), int64(len(tt.content)), "application/octet-stream", nil)
				}
				if err != nil {
					t.Fatal(err)
				}
				if hashes["md5"] != tt.md5 {
					t.Errorf("md5 = %q; want %q", hashes["md5"], tt.md5)
				}
				if hashes["sha1"] != tt.sha1 {
					t.Errorf("sha1 = %q; want %q", hashes["sha1"], tt.sha1)
				}
				if hashes["task_id"] != "test-task" {
					t.Errorf("task_id = %q; want test-task", hashes["task_id"])
				}
				if !finished {
					t.Error("quick upload was not finalized")
				}
			})
		}
	}
}
