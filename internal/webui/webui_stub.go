//go:build !embed_web

package webui

import (
	"errors"
	"io/fs"
)

var ErrNotEmbedded = errors.New("前端资源未嵌入，请先执行 pnpm build 并使用 -tags embed_web 编译")

func Embedded() bool {
	return false
}

func Dist() (fs.FS, error) {
	return nil, ErrNotEmbedded
}
