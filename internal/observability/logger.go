package observability

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Logger структурированный логер
type Logger struct {
	env string
}

func NewLogger(env string) *Logger {
	return &Logger{env: env}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.write("INFO", msg, fields...)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.write("ERROR", msg, fields...)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	if os.Getenv("DEBUG") == "1" {
		l.write("DEBUG", msg, fields...)
	}
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.write("WARN", msg, fields...)
}

func (l *Logger) write(level string, msg string, fields ...interface{}) {
	payload := map[string]interface{}{
		"ts":    time.Now().UTC().Format(time.RFC3339Nano),
		"level": level,
		"msg":   msg,
		"env":   l.env,
	}

	for i := 0; i < len(fields); i += 2 {
		key := "field"
		if k, ok := fields[i].(string); ok && k != "" {
			key = k
		}

		if i+1 < len(fields) {
			payload[key] = fields[i+1]
			continue
		}

		payload[key] = "(missing)"
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Println(level+":", msg, fields)
		return
	}

	log.Println(string(data))
}
