package stream

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

// StringData 写入 SSE 字符串数据
func StringData(c *gin.Context, str string) error {
	if c == nil || c.Writer == nil {
		return fmt.Errorf("context or writer is nil")
	}

	if c.Request.Context().Err() != nil {
		return fmt.Errorf("request context done: %w", c.Request.Context().Err())
	}

	_, err := c.Writer.Write([]byte("data: " + str + "\n\n"))
	if err != nil {
		return fmt.Errorf("write string data failed: %w", err)
	}
	c.Writer.Flush()
	return nil
}

// ObjectData 写入 SSE JSON 对象
func ObjectData(c *gin.Context, object interface{}) error {
	if object == nil {
		return fmt.Errorf("object is nil")
	}
	jsonData, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("marshal object failed: %w", err)
	}
	return StringData(c, string(jsonData))
}

// DoneData 写入 SSE 完成标记
func DoneData(c *gin.Context) error {
	return StringData(c, "done")
}
