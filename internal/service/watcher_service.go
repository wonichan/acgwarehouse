package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

type WatcherService struct {
	watcher      *fsnotify.Watcher
	scannerSvc   *ScannerService
	metadataSvc  *MetadataService
	imageRepo    repository.ImageRepository
	jobRepo      repository.JobRepository
	roots        []string
	debounceTime time.Duration

	mu     sync.Mutex
	timers map[string]*time.Timer
	closed bool
}

func NewWatcherService(scannerSvc *ScannerService, metadataSvc *MetadataService, imageRepo repository.ImageRepository, jobRepo repository.JobRepository, roots []string) (*WatcherService, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &WatcherService{
		watcher:      watcher,
		scannerSvc:   scannerSvc,
		metadataSvc:  metadataSvc,
		imageRepo:    imageRepo,
		jobRepo:      jobRepo,
		roots:        roots,
		debounceTime: 200 * time.Millisecond,
		timers:       make(map[string]*time.Timer),
	}, nil
}

func (w *WatcherService) Start(ctx context.Context) error {
	for _, root := range w.roots {
		if err := w.addRecursive(root); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			if err != nil {
				return err
			}
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			if err := w.handleEvent(event); err != nil {
				return err
			}
		}
	}
}

func (w *WatcherService) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	for _, timer := range w.timers {
		timer.Stop()
	}
	w.timers = map[string]*time.Timer{}
	_ = w.watcher.Close()
}

func (w *WatcherService) handleEvent(event fsnotify.Event) error {
	if event.Op&fsnotify.Create == fsnotify.Create {
		if stat, statErr := safeStat(event.Name); statErr == nil && stat.IsDir() {
			return w.addRecursive(event.Name)
		}
	}

	if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
		return nil
	}
	if !w.metadataSvc.IsImage(event.Name) {
		return nil
	}

	root := w.matchRoot(event.Name)
	if root == "" {
		return nil
	}

	w.scheduleImport(event.Name, root)
	return nil
}

func (w *WatcherService) scheduleImport(path, root string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if timer, ok := w.timers[path]; ok {
		timer.Stop()
	}
	w.timers[path] = time.AfterFunc(w.debounceTime, func() {
		_ = w.scannerSvc.importFile(path, root)
		w.mu.Lock()
		delete(w.timers, path)
		w.mu.Unlock()
	})
}

func (w *WatcherService) addRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if addErr := w.watcher.Add(path); addErr != nil {
				return addErr
			}
		}
		return nil
	})
}

func (w *WatcherService) matchRoot(path string) string {
	cleanPath := filepath.Clean(path)
	for _, root := range w.roots {
		cleanRoot := filepath.Clean(root)
		if sameOrChildPath(cleanRoot, cleanPath) {
			return cleanRoot
		}
	}
	return ""
}

type fileInfo interface {
	IsDir() bool
}

func safeStat(path string) (fileInfo, error) {
	return osStat(path)
}
