package logger

import "testing"

func TestContextLoggerMethods(t *testing.T) {
	var value Logger = NopLogger{}
	value.Info(nil, "test", "key", "value")
	value.Warn(nil, "test")
}
