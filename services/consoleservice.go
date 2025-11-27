package services

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// ConsoleLog 控制台日志条目
type ConsoleLog struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // INFO, WARN, ERROR
	Message   string    `json:"message"`
}

// ConsoleService 控制台日志服务
type ConsoleService struct {
	logs      []ConsoleLog
	mutex     sync.RWMutex
	maxLogs   int
	writer    *consoleWriter
	oldStdout *os.File
	oldStderr *os.File
}

// consoleWriter 自定义 writer，同时写入控制台和缓存
type consoleWriter struct {
	service *ConsoleService
	level   string
	output  io.Writer
}

func (w *consoleWriter) Write(p []byte) (n int, err error) {
	// 写入原始输出
	n, err = w.output.Write(p)

	// 添加到日志缓存
	w.service.addLog(w.level, string(p))

	return n, err
}

func NewConsoleService() *ConsoleService {
	cs := &ConsoleService{
		logs:    make([]ConsoleLog, 0, 1000),
		maxLogs: 1000, // 最多保留 1000 条日志
	}

	// 捕获标准输出和标准错误
	cs.captureStdout()

	return cs
}

// captureStdout 捕获标准输出和标准错误
func (cs *ConsoleService) captureStdout() {
	// 保存原始输出
	cs.oldStdout = os.Stdout
	cs.oldStderr = os.Stderr

	// 创建管道
	stdoutReader, stdoutWriter, _ := os.Pipe()
	stderrReader, stderrWriter, _ := os.Pipe()

	// 替换标准输出
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	log.SetOutput(stdoutWriter)

	// 启动 goroutine 读取管道内容
	go cs.readPipe(stdoutReader, "INFO", cs.oldStdout)
	go cs.readPipe(stderrReader, "ERROR", cs.oldStderr)
}

// readPipe 读取管道内容
func (cs *ConsoleService) readPipe(reader *os.File, level string, output *os.File) {
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(output, "读取管道失败: %v\n", err)
			}
			return
		}

		if n > 0 {
			msg := string(buf[:n])
			// 写入原始输出
			output.Write(buf[:n])
			// 添加到日志缓存
			cs.addLog(level, msg)
		}
	}
}

// addLog 添加日志到缓存
func (cs *ConsoleService) addLog(level, message string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	log := ConsoleLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	cs.logs = append(cs.logs, log)

	// 限制日志数量
	if len(cs.logs) > cs.maxLogs {
		cs.logs = cs.logs[len(cs.logs)-cs.maxLogs:]
	}
}

// GetLogs 获取所有日志
func (cs *ConsoleService) GetLogs() []ConsoleLog {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	// 返回副本
	result := make([]ConsoleLog, len(cs.logs))
	copy(result, cs.logs)
	return result
}

// GetRecentLogs 获取最近 N 条日志
func (cs *ConsoleService) GetRecentLogs(count int) []ConsoleLog {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	if count <= 0 {
		count = 100
	}

	if count > len(cs.logs) {
		count = len(cs.logs)
	}

	// 返回最后 N 条
	result := make([]ConsoleLog, count)
	copy(result, cs.logs[len(cs.logs)-count:])
	return result
}

// ClearLogs 清空日志
func (cs *ConsoleService) ClearLogs() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	cs.logs = make([]ConsoleLog, 0, 1000)
}
