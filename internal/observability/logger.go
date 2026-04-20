package observability

import (
	"log"
	"os"
)

// Logger структурированный логер
type Logger struct {
	// TODO: use zap or zerolog
}

func NewLogger(env string) *Logger {
	return &Logger{}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	log.Println("INFO:", msg, fields)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	log.Println("ERROR:", msg, fields)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	if os.Getenv("DEBUG") == "1" {
		log.Println("DEBUG:", msg, fields)
	}
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	log.Println("WARN:", msg, fields)
}
