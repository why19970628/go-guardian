package logger

import "testing"

func TestContextLoggerMethods(t *testing.T) {
	var value Logger = NopLogger{}
	value.InfoContext(nil, "test", "key", "value")
	value.WarnContext(nil, "test")
}
