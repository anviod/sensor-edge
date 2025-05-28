package mapping

import (
	"github.com/Knetic/govaluate"
)

// Mapper 物模型映射接口
type Mapper interface {
	Map(raw map[string]interface{}) (map[string]interface{}, error)
}

// EvalExpression 用于执行转换或报警表达式
func EvalExpression(expr string, value any) (any, error) {
	e, err := govaluate.NewEvaluableExpression(expr)
	if err != nil {
		return nil, err
	}
	result, err := e.Evaluate(map[string]interface{}{"value": value})
	return result, err
}
