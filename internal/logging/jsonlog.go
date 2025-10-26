package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type entry struct {
	Level   string         `json:"level"`
	Time    string         `json:"time"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func Log(level, msg string, fields map[string]any) {
	e := entry{Level: level, Time: time.Now().UTC().Format(time.RFC3339Nano), Message: msg, Fields: fields}
	b, _ := json.Marshal(e)
	fmt.Fprintln(os.Stdout, string(b))
}

func Info(msg string, fields map[string]any)  { Log("info", msg, fields) }
func Error(msg string, fields map[string]any) { Log("error", msg, fields) }
