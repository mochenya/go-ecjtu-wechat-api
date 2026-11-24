// Package render 把结构化数据投影成人类可读的文本视图与 JSON。本包不产生数据：
// 字段值一律来自域模型，这里只决定列集、宽度与对齐。
package render

import (
	"encoding/json"
	"io"
)

// JSON 把值以规范形态写进 w：两空格缩进加结尾换行。
func JSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}
