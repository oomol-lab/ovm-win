// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const logCount = 5

var cs = make([]*Context, 0, 10)

// New creates a new log file
func New(p, n string) (*Context, error) {
	c := &Context{
		path: p,
		name: n,
		syncWriter: syncWriter{
			m:    sync.Mutex{},
			file: nil,
		},
	}
	if err := c.createLog(); err != nil {
		return nil, err
	}

	cs = append(cs, c)

	return c, nil
}

// NewWithChildProcess creates a new log file with child process
//
// No files will be created, the latest log file will be used.
func NewWithChildProcess(p, n string) (*Context, error) {
	c := &Context{
		path:    p,
		name:    n,
		isChild: true,
		syncWriter: syncWriter{
			m:    sync.Mutex{},
			file: nil,
		},
	}
	if err := c.useExistLog(); err != nil {
		return nil, err
	}

	cs = append(cs, c)

	return c, nil
}

// NewOnlyCreate creates a new log file without any operation
//
// Only create a new log file, and return the path of the log file
func NewOnlyCreate(p, n string) (string, error) {
	c := &Context{
		path: p,
		name: n,
		syncWriter: syncWriter{
			m:    sync.Mutex{},
			file: nil,
		},
	}
	if err := c.createLog(); err != nil {
		return "", err
	}

	logPath := c.file.Name()

	c.Close()

	return logPath, nil
}

func CloseAll() {
	for _, c := range cs {
		_ = c.file.Sync()
		_ = c.file.Close()
	}
}

type syncWriter struct {
	m    sync.Mutex
	file *os.File
}

func (w *syncWriter) write(b []byte) (n int, err error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.file.Write(b)
}

type Context struct {
	path    string
	name    string
	isChild bool
	syncWriter
}

func (c *Context) createLog() error {
	count := logCount
	for i := count - 1; i > 0; i-- {
		logName := c.name
		if i > 1 {
			logName += "." + strconv.Itoa(i)
		}
		logPath := filepath.Join(c.path, logName+".log")

		if _, err := os.Stat(logPath); err == nil {
			err := os.Rename(logPath, filepath.Join(c.path, c.name+"."+strconv.Itoa(i+1)+".log"))
			if err != nil {
				return fmt.Errorf("cannot rename log file: %v", err)
			}
		}

		if i == 1 {
			f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_RDWR|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("cannot open log file: %v", err)
			}
			c.file = f
		}
	}
	return nil
}

func (c *Context) useExistLog() error {
	for i := 1; i <= logCount; i++ {
		logName := c.name
		if i > 1 {
			logName += "." + strconv.Itoa(i)
		}
		logPath := filepath.Join(c.path, logName+".log")

		if _, err := os.Stat(logPath); err != nil {
			continue
		}

		if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR, 0644); err != nil {
			return fmt.Errorf("cannot open log file: %v", err)
		} else {
			c.file = f
		}
		return nil
	}

	return fmt.Errorf("cannot find latest log file in: %s", c.path)
}

func (c *Context) NewWithAppendName(name string) (*Context, error) {
	return New(c.path, c.name+"-"+name)
}

func (c *Context) base(t, message string) {
	d := time.Now().Format("2006-01-02 15:04:05.000")
	tag := ""
	if t != "" {
		tag = fmt.Sprintf("[%s]: ", t)
	}

	if c.isChild {
		_, _ = c.write([]byte(fmt.Sprintf("%s [CHILD] %s%s\n", d, tag, message)))
	} else {
		_, _ = c.write([]byte(fmt.Sprintf("%s %s%s\n", d, tag, message)))
	}
}

func (c *Context) Raw(message string) {
	c.base("", message)
}

func (c *Context) Rawf(format string, args ...any) {
	c.Raw(fmt.Sprintf(format, args...))
}

func (c *Context) Info(message string) {
	c.base("INFO", message)
}

func (c *Context) Infof(format string, args ...any) {
	c.Info(fmt.Sprintf(format, args...))
}

func (c *Context) Warn(message string) {
	c.base("WARN", message)
	_ = c.file.Sync()
}

func (c *Context) Warnf(format string, args ...any) {
	c.Warn(fmt.Sprintf(format, args...))
}

func (c *Context) Error(message string) error {
	c.base("ERROR", message)
	_ = c.file.Sync()
	return fmt.Errorf(message)
}

func (c *Context) Errorf(format string, args ...any) error {
	return c.Error(fmt.Sprintf(format, args...))
}

func (c *Context) Close() {
	_ = c.file.Close()

	for i, context := range cs {
		if context == c {
			cs = append(cs[:i], cs[i+1:]...)
			break
		}
	}
}
