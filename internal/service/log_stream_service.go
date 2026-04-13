package service

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/logger"
)

const (
	logStreamSubscriberBuffer = 16
	logStreamPollInterval     = 250 * time.Millisecond
	logStreamBufferLines      = 200
)

type LogSource string

const (
	LogSourceGo LogSource = "go"
)

type LogEvent struct {
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Payload   string    `json:"payload"`
	Severity  string    `json:"severity,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type LogStreamService struct {
	mu          sync.Mutex
	subscribers map[LogSource]map[int]chan LogEvent
	nextID      int
	goLogPath   string
	cancel      context.CancelFunc
	buffers     map[LogSource]*ringBuffer
	running     map[LogSource]bool
	wg          sync.WaitGroup
}

type ringBuffer struct {
	lines []string
	cap   int
	head  int
	size  int
}

func NewLogStreamService(goLogPath string) *LogStreamService {
	return &LogStreamService{
		subscribers: map[LogSource]map[int]chan LogEvent{
			LogSourceGo: make(map[int]chan LogEvent),
		},
		goLogPath: strings.TrimSpace(goLogPath),
		buffers: map[LogSource]*ringBuffer{
			LogSourceGo: newRingBuffer(logStreamBufferLines),
		},
		running: map[LogSource]bool{
			LogSourceGo: false,
		},
	}
}

func (s *LogStreamService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	logger.Infof("[service] LogStreamService started: go_log_path=%s", s.goLogPath)

	for _, source := range []LogSource{LogSourceGo} {
		initialOffset := s.seedSource(source)
		s.mu.Lock()
		s.running[source] = true
		s.wg.Add(1)
		s.mu.Unlock()
		go s.watchSource(runCtx, source, initialOffset)
	}
}

func (s *LogStreamService) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	if cancel == nil {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	cancel()
	logger.Infof("[service] LogStreamService stopped")
	s.wg.Wait()

	s.mu.Lock()
	s.cancel = nil
	s.mu.Unlock()
}

func (s *LogStreamService) Subscribe(source LogSource, tail int) (<-chan LogEvent, func()) {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	ch := make(chan LogEvent, logStreamSubscriberBuffer)
	if s.subscribers[source] == nil {
		s.subscribers[source] = make(map[int]chan LogEvent)
	}
	s.subscribers[source][id] = ch
	snapshotLines := s.snapshotLinesLocked(source, tail)
	s.mu.Unlock()

	if len(snapshotLines) > 0 {
		sendLogEvent(ch, LogEvent{Type: "snapshot", Source: string(source), Payload: marshalLogSnapshot(snapshotLines), Timestamp: time.Now().UTC()})
	} else if s.sourceMissing(source) {
		sendLogEvent(ch, LogEvent{Type: "status", Source: string(source), Payload: "log file not found", Timestamp: time.Now().UTC()})
	}

	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		subscriber, ok := s.subscribers[source][id]
		if !ok {
			return
		}
		delete(s.subscribers[source], id)
		close(subscriber)
	}

	return ch, unsubscribe
}

func (s *LogStreamService) Broadcast(source LogSource, event LogEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, subscriber := range s.subscribers[source] {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (s *LogStreamService) watchSource(ctx context.Context, source LogSource, offset int64) {
	defer s.wg.Done()
	defer func() {
		s.mu.Lock()
		s.running[source] = false
		s.mu.Unlock()
	}()

	ticker := time.NewTicker(logStreamPollInterval)
	defer ticker.Stop()

	var missingNotified bool
	for {
		status, nextOffset, missing := s.pollSource(source, offset)
		if missing {
			offset = 0
			if !missingNotified {
				s.Broadcast(source, LogEvent{Type: "status", Source: string(source), Payload: "log file not found", Timestamp: time.Now().UTC()})
				missingNotified = true
			}
		} else {
			missingNotified = false
			offset = nextOffset
			if status == watchStatusTruncated {
				lines := s.bufferLines(source, 0)
				s.Broadcast(source, LogEvent{Type: "snapshot", Source: string(source), Payload: marshalLogSnapshot(lines), Timestamp: time.Now().UTC()})
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type watchStatus string

const (
	watchStatusNoop      watchStatus = "noop"
	watchStatusAppended  watchStatus = "appended"
	watchStatusTruncated watchStatus = "truncated"
)

func (s *LogStreamService) pollSource(source LogSource, offset int64) (watchStatus, int64, bool) {
	path := s.pathForSource(source)
	if path == "" {
		return watchStatusNoop, offset, true
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return watchStatusNoop, 0, true
		}
		return watchStatusNoop, offset, false
	}

	truncated := info.Size() < offset
	if truncated {
		offset = 0
		s.resetBuffer(source)
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return watchStatusNoop, 0, true
		}
		return watchStatusNoop, offset, false
	}
	defer file.Close()

	if _, err := file.Seek(offset, 0); err != nil {
		return watchStatusNoop, offset, false
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if line != "" {
				severity := detectSeverity(line)
				s.appendLine(source, line)
				if !truncated {
					s.Broadcast(source, LogEvent{Type: "line", Source: string(source), Payload: line, Severity: severity, Timestamp: time.Now().UTC()})
				}
			}
		}
		if err != nil {
			break
		}
	}

	currentOffset, err := file.Seek(0, 1)
	if err != nil {
		currentOffset = info.Size()
	}
	if truncated {
		return watchStatusTruncated, currentOffset, false
	}
	if currentOffset > offset {
		return watchStatusAppended, currentOffset, false
	}
	return watchStatusNoop, currentOffset, false
}

func (s *LogStreamService) seedSource(source LogSource) int64 {
	path := s.pathForSource(source)
	if path == "" {
		return 0
	}

	info, err := os.Stat(path)
	if err != nil {
		return 0
	}

	lines, err := readLastLinesFromPath(path, logStreamBufferLines)
	if err != nil {
		return 0
	}

	s.resetBuffer(source)
	for _, line := range lines {
		s.appendLine(source, line)
	}
	return info.Size()
}

func (s *LogStreamService) snapshotLinesLocked(source LogSource, tail int) []string {
	if buffer := s.buffers[source]; buffer != nil && buffer.size > 0 {
		return buffer.last(tail)
	}
	return readLastLines(s.pathForSource(source), tail)
}

func (s *LogStreamService) bufferLines(source LogSource, tail int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.buffers[source] == nil {
		return nil
	}
	if tail <= 0 {
		return s.buffers[source].last(s.buffers[source].size)
	}
	return s.buffers[source].last(tail)
}

func (s *LogStreamService) appendLine(source LogSource, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.buffers[source] == nil {
		s.buffers[source] = newRingBuffer(logStreamBufferLines)
	}
	s.buffers[source].add(line)
}

func (s *LogStreamService) resetBuffer(source LogSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.buffers[source] == nil {
		s.buffers[source] = newRingBuffer(logStreamBufferLines)
		return
	}
	s.buffers[source].clear()
}

func (s *LogStreamService) sourceMissing(source LogSource) bool {
	path := s.pathForSource(source)
	if path == "" {
		return true
	}
	_, err := os.Stat(path)
	return err != nil && os.IsNotExist(err)
}

func (s *LogStreamService) pathForSource(source LogSource) string {
	switch source {
	case LogSourceGo:
		return s.goLogPath
	default:
		return ""
	}
}

func newRingBuffer(capacity int) *ringBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &ringBuffer{lines: make([]string, capacity), cap: capacity}
}

func (r *ringBuffer) add(line string) {
	if r == nil || r.cap == 0 {
		return
	}
	idx := (r.head + r.size) % r.cap
	if r.size == r.cap {
		idx = r.head
		r.head = (r.head + 1) % r.cap
	} else {
		r.size++
	}
	r.lines[idx] = line
}

func (r *ringBuffer) last(n int) []string {
	if r == nil || r.size == 0 {
		return nil
	}
	if n <= 0 || n > r.size {
		n = r.size
	}
	start := r.size - n
	out := make([]string, 0, n)
	for i := start; i < r.size; i++ {
		out = append(out, r.lines[(r.head+i)%r.cap])
	}
	return out
}

func (r *ringBuffer) clear() {
	if r == nil {
		return
	}
	r.head = 0
	r.size = 0
	for i := range r.lines {
		r.lines[i] = ""
	}
}

func detectSeverity(line string) string {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "level=fatal") || strings.Contains(lower, "level=panic") || strings.Contains(lower, "level=error") {
		return "error"
	}
	if strings.Contains(lower, "level=warn") || strings.Contains(lower, "level=warning") {
		return "warning"
	}
	if strings.Contains(lower, "level=info") || strings.Contains(lower, "level=debug") {
		return "normal"
	}
	for _, kw := range []string{"error", "fatal", "panic", "traceback", "exception", "failed"} {
		if strings.Contains(lower, kw) {
			return "error"
		}
	}
	for _, kw := range []string{"warn", "warning"} {
		if strings.Contains(lower, kw) {
			return "warning"
		}
	}
	return "normal"
}

func readLastLines(path string, tail int) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	lines, err := readLastLinesFromPath(path, tail)
	if err != nil {
		return nil
	}
	return lines
}

func readLastLinesFromPath(path string, n int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return nil, nil
	}

	const maxTailBytes = 64 * 1024
	readSize := maxTailBytes
	if int64(readSize) > size {
		readSize = int(size)
	}

	buf := make([]byte, readSize)
	if _, err := file.ReadAt(buf, size-int64(readSize)); err != nil {
		return nil, err
	}

	lines := splitLogLines(string(buf))
	if n <= 0 || n >= len(lines) {
		return lines, nil
	}
	return append([]string(nil), lines[len(lines)-n:]...), nil
}

func splitLogLines(content string) []string {
	return strings.FieldsFunc(content, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}

func marshalLogSnapshot(lines []string) string {
	payload, err := json.Marshal(lines)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func sendLogEvent(ch chan LogEvent, event LogEvent) {
	select {
	case ch <- event:
	default:
	}
}
