package routes

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterDownloadRoutes 注册 /downloads/*：从 dir 目录静态提供安装包等文件。
//
// 用于新手引导里的 CC Switch 自托管下载（国内访问 GitHub 很慢）。约定目录结构：
//
//	<dir>/cc-switch/manifest.json   版本与各文件的大小、sha256（由 tools/sync_ccswitch.py 生成）
//	<dir>/cc-switch/<安装包文件>
//
// dir 为空表示未启用，所有请求返回 404（前端据此回退到 GitHub 下载）。
// 只提供普通文件：不列目录、不跟随越出 dir 的路径、不提供以 . 开头的文件。
func RegisterDownloadRoutes(r *gin.Engine, dir string) {
	root := strings.TrimSpace(dir)
	handle := func(c *gin.Context) {
		if root == "" {
			c.Status(http.StatusNotFound)
			return
		}
		full, ok := resolveDownloadPath(root, c.Param("filepath"))
		if !ok {
			c.Status(http.StatusNotFound)
			return
		}
		info, err := os.Stat(full)
		if err != nil || !info.Mode().IsRegular() {
			c.Status(http.StatusNotFound)
			return
		}
		if strings.HasSuffix(full, ".json") {
			// manifest 要能很快反映新版本。
			c.Header("Cache-Control", "no-cache")
		} else {
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("Content-Disposition", `attachment; filename="`+filepath.Base(full)+`"`)
		}
		c.Header("X-Content-Type-Options", "nosniff")
		c.File(full)
	}
	// 下载工具常先发 HEAD 探测大小，两种方法都要支持。
	r.GET("/downloads/*filepath", handle)
	r.HEAD("/downloads/*filepath", handle)
}

// resolveDownloadPath 把请求路径安全地映射到 root 下的文件；越界或含隐藏段时返回 false。
func resolveDownloadPath(root, requested string) (string, bool) {
	clean := filepath.Clean("/" + strings.TrimPrefix(requested, "/"))
	if clean == "/" {
		return "", false
	}
	for _, seg := range strings.Split(strings.Trim(clean, "/"), "/") {
		if seg == "" || strings.HasPrefix(seg, ".") {
			return "", false
		}
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	full := filepath.Join(absRoot, filepath.FromSlash(clean))
	// 解析符号链接后仍须在 root 内，避免通过链接读到目录之外的文件。
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", false
	}
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", false
	}
	if resolved != resolvedRoot && !strings.HasPrefix(resolved, resolvedRoot+string(os.PathSeparator)) {
		return "", false
	}
	return resolved, true
}
