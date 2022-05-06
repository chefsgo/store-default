package file_default

import (
	"github.com/chefsgo/file"
)

func Driver() file.Driver {
	return &defaultDriver{}
}

func init() {
	file.Register("default", Driver())
}
