package files

import (
	"context"

	"github.com/anfragment/zen/internal/cfg"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type FileImport struct {
	ctx context.Context
	cfg *cfg.Config
}

func NewFileImport() *FileImport {
	return &FileImport{}
}

func (i *FileImport) Init(ctx context.Context, cfg *cfg.Config) {
	i.ctx = ctx
	i.cfg = cfg
}

func (i *FileImport) ImportFilterList() error {
	filePath, err := runtime.OpenFileDialog(i.ctx, runtime.OpenDialogOptions{
		Title: "Import Custom Filter Lists",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON", Pattern: "*.json"},
		},
	})

	if err != nil {
		return err
	}

	return i.cfg.ImportFilterList(filePath)
}
