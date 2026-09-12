package logger

import "testing"

func TestNopLoggerImplementsContract(t *testing.T) {
	var _ Logger = NopLogger{}
	NopLogger{}.Infow("test", "key", "value")
}
