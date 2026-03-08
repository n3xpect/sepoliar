package logger

import (
	"encoding/json"
	"time"
)

type Field struct {
	Key   string
	Value any
}

func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func ToJSON(key, value string) Field {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(value), &jsonData); err == nil {
		return Field{Key: key, Value: jsonData}
	}
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

func ErrJSON(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}

	errMsg := err.Error()

	var jsonData map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(errMsg), &jsonData); jsonErr == nil {
		return Field{Key: "error", Value: jsonData}
	}

	return Field{Key: "error", Value: errMsg}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}
