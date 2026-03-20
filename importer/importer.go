package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"linebackerr/filemanager"
	"linebackerr/matcher"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Mode string

const (
	ModeMove     Mode = "move"
	ModeHardlink Mode = "hardlink"
	ModeSymlink  Mode = "symlink"
	ModeCopy     Mode = "copy"
)

type Options struct {
	LibraryRoot string
	Mode        Mode
}

type ImportResult struct {
	SourcePath      string
	DestinationPath string
	RelativePath    string
	Mode            Mode
	AlreadyPresent  bool
}

type Service struct {
	mode        Mode
	fileManager *filemanager.Manager
}

func NewService(options Options) (*Service, error) {
	manager, err := filemanager.New(options.LibraryRoot)
	if err != nil {
		return nil, err
	}

	mode := options.Mode
	if mode == "" {
		mode = ModeMove
	}
	switch mode {
	case ModeMove, ModeHardlink, ModeSymlink, ModeCopy:
	default:
		return nil, fmt.Errorf("unsupported importer mode: %s", mode)
	}

	return &Service{fileManager: manager, mode: mode}, nil
}

func (s *Service) ImportFile(ctx context.Context, match matcher.Match, sourcePath string) (ImportResult, error) {
	if s == nil {
		return ImportResult{}, errors.New("nil importer service")
	}
	if err := ctx.Err(); err != nil {
		return ImportResult{}, err
	}

	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if sourcePath == "" {
		return ImportResult{}, errors.New("source path is required")
	}
	if _, err := os.Stat(sourcePath); err != nil {
		return ImportResult{}, fmt.Errorf("stat source file: %w", err)
	}

	placement, err := s.fileManager.PrepareImportTarget(match, sourcePath)
	if err != nil {
		return ImportResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(placement.AbsolutePath), 0o755); err != nil {
		return ImportResult{}, fmt.Errorf("create library directories: %w", err)
	}

	if sameFile(sourcePath, placement.AbsolutePath) {
		return ImportResult{
			SourcePath:      sourcePath,
			DestinationPath: placement.AbsolutePath,
			RelativePath:    placement.RelativePath,
			Mode:            s.mode,
			AlreadyPresent:  true,
		}, nil
	}

	if err := s.placeFile(sourcePath, placement.AbsolutePath); err != nil {
		return ImportResult{}, err
	}

	return ImportResult{
		SourcePath:      sourcePath,
		DestinationPath: placement.AbsolutePath,
		RelativePath:    placement.RelativePath,
		Mode:            s.mode,
	}, nil
}

func (s *Service) ImportFiles(ctx context.Context, match matcher.Match, sourcePaths []string) ([]ImportResult, error) {
	results := make([]ImportResult, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		result, err := s.ImportFile(ctx, match, sourcePath)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) placeFile(sourcePath, destinationPath string) error {
	switch s.mode {
	case ModeMove:
		if err := os.Rename(sourcePath, destinationPath); err == nil {
			return nil
		}
		if err := copyFile(sourcePath, destinationPath); err != nil {
			return err
		}
		if err := os.Remove(sourcePath); err != nil {
			return fmt.Errorf("remove source after move fallback: %w", err)
		}
		return nil
	case ModeHardlink:
		if err := os.Link(sourcePath, destinationPath); err != nil {
			return fmt.Errorf("create hardlink: %w", err)
		}
		return nil
	case ModeSymlink:
		if err := os.Symlink(sourcePath, destinationPath); err != nil {
			return fmt.Errorf("create symlink: %w", err)
		}
		return nil
	case ModeCopy:
		if err := copyFile(sourcePath, destinationPath); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported importer mode: %s", s.mode)
	}
}

func sameFile(sourcePath, destinationPath string) bool {
	src, err := os.Stat(sourcePath)
	if err != nil {
		return false
	}
	dst, err := os.Stat(destinationPath)
	if err != nil {
		return false
	}
	return os.SameFile(src, dst)
}

func copyFile(sourcePath, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer source.Close()

	info, err := source.Stat()
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}

	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open destination file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(destinationPath, info.Mode().Perm())
	}

	return nil
}
