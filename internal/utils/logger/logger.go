package logger

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
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

	instanceMu     sync.Mutex
	instances      []*FSLogger
	instancesLogs  []string
	instancesModes []string
)

// TODO: пофиксить утечку файловых дескрипторов
func GetLogger(logPath, mode string) (*FSLogger, error) {
	instanceMu.Lock()
	defer instanceMu.Unlock()

	if mode != "DEBUG" && mode != "PROD" {
		return nil, InvalidModeError
	}
	for i, instance := range instances {
		if instancesLogs[i] == logPath && instancesModes[i] == mode {
			return instance, nil
		}
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

	instance := &FSLogger{
		logMode: mode,
		file:    file,
	}

	instances = append(instances, instance)

	instancesLogs = append(instancesLogs, logPath)
	instancesModes = append(instancesModes, mode)

	return instance, nil
}

func (l *FSLogger) write(text string, needPrettify bool) error {
	l.writeMu.Lock()
	defer l.writeMu.Unlock()

	if l.file == nil {
		return errors.New("logger is not initialized")
	}
	if needPrettify {
		text = prettifyPayloadForLog(text)
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

func (l *FSLogger) WritePrettified(text string) error {
	return l.write(text, true)
}
func (l *FSLogger) WriteNonPrettified(text string) error {
	return l.write(text, false)
}
func prettifyPayloadForLog(payload string) string {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return payload
	}

	var jsonBuf bytes.Buffer
	if json.Valid([]byte(trimmed)) && json.Indent(&jsonBuf, []byte(trimmed), "", "  ") == nil {
		return jsonBuf.String()
	}

	if strings.HasPrefix(trimmed, "<") {
		if pretty, err := prettifyXML(trimmed); err == nil {
			return pretty
		}
	}

	return payload
}

func prettifyXML(raw string) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(raw))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if err := enc.EncodeToken(tok); err != nil {
			return "", err
		}
	}

	if err := enc.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
