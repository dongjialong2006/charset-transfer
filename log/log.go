package log

import (
	"charset-transfer/file"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	formatter "github.com/x-cray/logrus-prefixed-formatter"
)

const (
	LOCAL  = "local"
	REMOTE = "remote"
)

const (
	MAXLOGSNUM = 6
)

type Log struct{}

func (l *Log) Write(p []byte) (n int, err error) {
	return 0, nil
}

func New(name string) *logrus.Entry {
	return logrus.WithField("model", name)
}

func newWriter(level logrus.Level, writer *rotatelogs.RotateLogs) lfshook.WriterMap {
	var handles = make(lfshook.WriterMap)
	switch level {
	case 0:
		logrus.SetLevel(logrus.PanicLevel)
		handles[logrus.PanicLevel] = writer
	case 1:
		logrus.SetLevel(logrus.FatalLevel)
		handles[logrus.PanicLevel] = writer
		handles[logrus.FatalLevel] = writer
	case 2:
		logrus.SetLevel(logrus.ErrorLevel)
		handles[logrus.PanicLevel] = writer
		handles[logrus.FatalLevel] = writer
		handles[logrus.ErrorLevel] = writer
	case 3:
		logrus.SetLevel(logrus.WarnLevel)
		handles[logrus.PanicLevel] = writer
		handles[logrus.FatalLevel] = writer
		handles[logrus.ErrorLevel] = writer
		handles[logrus.WarnLevel] = writer
	case 4:
		logrus.SetLevel(logrus.InfoLevel)
		handles[logrus.PanicLevel] = writer
		handles[logrus.FatalLevel] = writer
		handles[logrus.ErrorLevel] = writer
		handles[logrus.WarnLevel] = writer
		handles[logrus.InfoLevel] = writer
	case 5:
		logrus.SetLevel(logrus.DebugLevel)
		handles[logrus.PanicLevel] = writer
		handles[logrus.FatalLevel] = writer
		handles[logrus.ErrorLevel] = writer
		handles[logrus.WarnLevel] = writer
		handles[logrus.InfoLevel] = writer
		handles[logrus.DebugLevel] = writer
	default:
		return nil
	}

	return handles
}

func LocalFileSystemLogger(ctx context.Context, path string, level logrus.Level) error {
	if "" == path {
		return fmt.Errorf("log psth is empty.")
	}

	err := file.CreatePath(path)
	if nil != err {
		return err
	}

	path = fmt.Sprintf("%s_%d", path, os.Getpid())

	writer, err := rotatelogs.New(
		path+".%Y_%m_%d",
		rotatelogs.WithMaxAge(MAXLOGSNUM*24*time.Hour), // 文件最大保存时间
		rotatelogs.WithRotationTime(24*time.Hour),      // 日志切割时间间隔
		rotatelogs.WithRotationCount(MAXLOGSNUM),
	)
	if err != nil {
		return err
	}

	logrus.SetOutput(&Log{})

	lfHook := lfshook.NewHook(newWriter(level, writer), &formatter.TextFormatter{
		TimestampFormat:  "2006-01-02 15:04:05.0000",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	logrus.AddHook(lfHook)
	return nil
}

func RemoteFileSystemLogger(ctx context.Context, addr string) error {
	// connect log store

	// receive log and send to store
	return nil
}
