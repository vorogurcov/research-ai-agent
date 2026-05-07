package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FSLogger struct {
	logMode string
	file    *os.File
	writeMu sync.Mutex
}

var (
	InvalidModeError = errors.New("logger mode should be 'DEBUG' or 'PROD'")

	instanceMu   sync.Mutex
	instance     *FSLogger
	instanceLog  string
	instanceMode string
)

func (l *FSLogger) GetLogger(logPath, mode string) (*FSLogger, error) {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	if mode != "DEBUG" && mode != "PROD" {
		return nil, InvalidModeError
	}

	if instance != nil && instanceLog == logPath && instanceMode == mode {
		return instance, nil
	}

	if instance != nil && instance.file != nil {
		_ = instance.file.Close()
		instance = nil
	}

	if logPath == "" {
		return nil, errors.New("logPath must not be empty")
	}

	if dir := filepath.Dir(logPath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	instance = &FSLogger{
		logMode: mode,
		file:    file,
	}

	instanceLog = logPath
	instanceMode = mode

	return instance, nil
}

func (l *FSLogger) Write(text string) error {
	l.writeMu.Lock()
	defer l.writeMu.Unlock()

	if l.file == nil {
		return errors.New("logger is not initialized")
	}

	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}

	fullString := "[" + time.Now().Format(time.DateTime) + "] " + text
	_, err := l.file.WriteString(fullString)
	if err != nil {
		return err
	}
	return nil
}
