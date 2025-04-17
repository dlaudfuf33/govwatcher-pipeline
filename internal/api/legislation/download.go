package legislation

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Xlsx 파일 다운로드 함수
func SaveResponseToFile(resp *http.Response, path string) error {
    err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
    if err != nil {
        return err
    }
    out, err := os.Create(path)
    if err != nil {
        return err
    }
    defer out.Close()
    _, err = io.Copy(out, resp.Body)
    return err
}
