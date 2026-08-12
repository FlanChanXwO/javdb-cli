package pipeline

// BatchFunc 处理一个输入项并返回输出信封；错误会转成原位错误信封。
type BatchFunc func(Envelope) (Envelope, error)

// RunBatch 按输入顺序处理全部项：单项失败输出原位错误信封并继续，成功项不
// 丢失；返回失败数与输出错误。调用方在所有输出完成后按失败数决定非零退出。
func RunBatch(writer *Writer, inputs []Envelope, command string, fn BatchFunc) (failures int, outputErr error) {
	for _, input := range inputs {
		output, err := fn(input)
		if err != nil {
			failures++
			output = ErrorEnvelope(input, command, "batch", "item", err.Error())
		}
		if writeErr := writer.Write(output); writeErr != nil {
			return failures, writeErr
		}
	}
	return failures, nil
}
