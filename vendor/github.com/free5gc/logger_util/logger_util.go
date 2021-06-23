package logger_util

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type FileHook struct {
	file      *os.File
	flag      int
	chmod     os.FileMode
	formatter *logrus.TextFormatter
}

func NewFileHook(file string, flag int, chmod os.FileMode) (*FileHook, error) {
	plainFormatter := &logrus.TextFormatter{DisableColors: true}
	logFile, err := os.OpenFile(file, flag, chmod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to write file on filehook %v", err)
		return nil, err
	}

	return &FileHook{logFile, flag, chmod, plainFormatter}, err
}

// Fire event
func (hook *FileHook) Fire(entry *logrus.Entry) error {
	var line string
	if plainformat, err := hook.formatter.Format(entry); err != nil {
		log.Printf("Formatter error: %+v", err)
		return err
	} else {
		line = string(plainformat)
	}
	if _, err := hook.file.WriteString(line); err != nil {
		fmt.Fprintf(os.Stderr, "unable to write file on filehook(entry.String)%v", err)
		return err
	}

	return nil
}

func (hook *FileHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

//The Middleware will write the Gin logs to logrus.
func ginToLogrus(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Infof("| %3d | %15s | %-7s | %s | %s",
			statusCode, clientIP, method, path, errorMessage)
	}
}

//NewGinWithLogrus - returns an Engine instance with the ginToLogrus and Recovery middleware already attached.
func NewGinWithLogrus(log *logrus.Entry) *gin.Engine {
	engine := gin.New()
	engine.Use(ginToLogrus(log), gin.Recovery())
	return engine
}
