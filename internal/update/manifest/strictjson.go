package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// canonicalJSON 编码为不带尾随换行的紧凑 JSON；字段顺序由 struct 声明顺序
// 固定，因此对同一值产生的字节稳定，签名可复现。
func canonicalJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

// decodeStrictJSON 严格解码单个 JSON 文档：拒绝重复对象键、未知结构之后的
// 尾随内容与语法错误，防止歧义编码被签名为不同语义。
func decodeStrictJSON(raw []byte, value any) error {
	if err := checkDuplicateKeys(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if decoder.More() {
		return errors.New("trailing data after JSON document")
	}
	return nil
}

// checkDuplicateKeys 遍历 JSON 文档，拒绝同一对象中重复的键。
func checkDuplicateKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	type objectState struct {
		seen      map[string]bool
		expectKey bool
	}
	var stack []*objectState
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		delimiter, isDelimiter := token.(json.Delim)
		if isDelimiter {
			switch delimiter {
			case '{':
				stack = append(stack, &objectState{seen: map[string]bool{}, expectKey: true})
			case '[':
				stack = append(stack, &objectState{seen: nil})
			case '}', ']':
				if len(stack) == 0 {
					return errors.New("unexpected closing delimiter")
				}
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if len(stack) == 0 {
			continue
		}
		top := stack[len(stack)-1]
		if top.seen == nil {
			// 数组内元素不参与键检查。
			continue
		}
		text, isString := token.(string)
		if isString && top.expectKey {
			if top.seen[text] {
				return fmt.Errorf("duplicate JSON key %q", text)
			}
			top.seen[text] = true
			top.expectKey = false
			continue
		}
		top.expectKey = true
	}
	return nil
}
