package test

import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strings"
	"sync/atomic"
	"testing"
)

var (
	current int64
)

type Named interface {
	name() string
}

type Ctx interface {
	Get(key string) (value any, exist bool)
	Set(key string, value any)
	Name() string
}

type RecoverFuncBuilder struct {
	t      *testing.T
	err    error
	doing  []func(ctx Ctx) (any, error)
	onSuc  []func(ctx Ctx) (any, error)
	onFail []func(ctx Ctx) (any, error)
}

type Person struct {
	Name string
	Age  int
}

func (p *Person) name() string {
	return p.Name
}

func resetCurrent() {
	atomic.StoreInt64(&current, 0)
}

func SetCtx() func(ctx Ctx) (any, error) {
	return SetCtx0("")
}

func SetCtx0(prefix string) func(ctx Ctx) (any, error) {
	return func(ctx Ctx) (any, error) {
		ctx.Set(prefix+"int", 1)
		ctx.Set(prefix+"int8", int8(2))
		ctx.Set(prefix+"int16", int16(3))
		ctx.Set(prefix+"int32", int32(4))
		ctx.Set(prefix+"int64", int64(5))
		ctx.Set(prefix+"uint", uint(6))
		ctx.Set(prefix+"uint8", uint8(7))
		ctx.Set(prefix+"uint16", uint16(8))
		ctx.Set(prefix+"uint32", uint32(9))
		ctx.Set(prefix+"uint64", uint64(10))
		ctx.Set(prefix+"float32", float32(11.01))
		ctx.Set(prefix+"float64", float64(12.01))
		ctx.Set(prefix+"bool", true)
		ctx.Set(prefix+"string", ctx.Name())
		ctx.Set(prefix+"password", ctx.Name())
		ctx.Set(prefix+"pwd", ctx.Name())
		ctx.Set(prefix+"Person", Person{ctx.Name(), 11})
		ctx.Set(prefix+"*Person", &Person{ctx.Name(), 11})
		var named Named
		named = &Person{ctx.Name(), 11}
		ctx.Set(prefix+"Named", named)
		return nil, nil
	}
}

func CheckCtx(preview string, notCheckResult ...int) func(ctx Ctx) (any, error) {
	return CheckCtx0(preview, "", notCheckResult...)
}

func CheckCtx0(preview, prefix string, notCheckResult ...int) func(ctx Ctx) (any, error) {
	return func(ctx Ctx) (any, error) {
		if step, ok := ctx.(flow.Step); ok && len(notCheckResult) == 0 {
			if res, exist := step.Result(preview); exist {
				if res.(string) != preview {
					panic(fmt.Sprintf("Step[%s] check Result[%s] failed, result=[%#v]", ctx.Name(), preview, res))
				}
			} else {
				panic(fmt.Sprintf("Step[%s] check Result[%s] failed, result not exist", ctx.Name(), preview))
			}
		}
		if value, exist := ctx.Get(prefix + "int"); exist {
			if value.(int) != 1 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int failed, int=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int failed, int=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int8"); exist {
			if value.(int8) != 2 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int8 failed, int8=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int8 failed, int8=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int16"); exist {
			if value.(int16) != 3 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int16 failed, int16=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int16 failed, int16=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int32"); exist {
			if value.(int32) != 4 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int32 failed, int32=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int32 failed, int32=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "int64"); exist {
			if value.(int64) != 5 {
				return ctx.Name(), fmt.Errorf("Step[%s] check int64 failed, int64=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check int64 failed, int64=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint"); exist {
			if value.(uint) != 6 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint failed, uint=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint failed, uint=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint8"); exist {
			if value.(uint8) != 7 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint8 failed, uint8=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint8 failed, uint8=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint16"); exist {
			if value.(uint16) != 8 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint16 failed, uint16=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint16 failed, uint16=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint32"); exist {
			if value.(uint32) != 9 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint32 failed, uint32=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint32 failed, uint32=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "uint64"); exist {
			if value.(uint64) != 10 {
				return ctx.Name(), fmt.Errorf("Step[%s] check uint64 failed, uint64=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check uint64 failed, uint64=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "float32"); exist {
			if value.(float32) != 11.01 {
				return ctx.Name(), fmt.Errorf("Step[%s] check float32 failed, float32=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check float32 failed, float32=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "float64"); exist {
			if value.(float64) != 12.01 {
				return ctx.Name(), fmt.Errorf("Step[%s] check float64 failed, float64=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check float64 failed, float64=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "bool"); exist {
			if value.(bool) != true {
				return ctx.Name(), fmt.Errorf("Step[%s] check bool failed, bool=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check bool failed, bool=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "Person"); exist {
			if value.(Person).Name != preview || value.(Person).Age != 11 {
				return ctx.Name(), fmt.Errorf("Step[%s] check Person failed, Person=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check Person failed, Person=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "*Person"); exist {
			if value.(*Person).Name != preview || value.(*Person).Age != 11 {
				return ctx.Name(), fmt.Errorf("Step[%s] check *Person failed, *Person=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check *Person failed, *Person=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "Named"); exist {
			if value.(*Person).Name != preview || value.(*Person).Age != 11 {
				return ctx.Name(), fmt.Errorf("Step[%s] check Named failed, Named=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check Named failed, Named=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "string"); exist {
			if value.(string) != preview {
				return ctx.Name(), fmt.Errorf("Step[%s] check string failed, string=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check string failed, string=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "password"); exist {
			if value.(string) != preview {
				return ctx.Name(), fmt.Errorf("Step[%s] check password failed, password=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check password failed, password=[%#v]", ctx.Name(), value)
		}
		if value, exist := ctx.Get(prefix + "pwd"); exist {
			if value.(string) != preview {
				return ctx.Name(), fmt.Errorf("Step[%s] check pwd failed, pwd=[%#v]", ctx.Name(), value)
			}
		} else {
			return ctx.Name(), fmt.Errorf("Step[%s] check pwd failed, pwd=[%#v]", ctx.Name(), value)
		}
		return ctx.Name(), nil
	}
}

func CheckResult(t *testing.T, check int64, statuses ...*flow.StatusEnum) func(flow.WorkFlow) (keepOn bool, err error) {
	return func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		ss := make([]string, len(statuses))
		for i, status := range statuses {
			ss[i] = status.Message()
		}
		t.Logf("start check, expected current=%d, status include %s", check, strings.Join(ss, ","))
		if current != check {
			t.Errorf("execute %d step, but current = %d\n", check, current)
		}
		for _, status := range statuses {
			if status == flow.Success && !workFlow.Success() {
				t.Errorf("WorkFlow executed failed\n")
			}
			//if status == flow.Timeout {
			//	time.Sleep(50 * time.Millisecond)
			//}
			if !workFlow.Has(status) {
				t.Errorf("workFlow has not %s status\n", status.Message())
			}
		}
		t.Logf("status expalin=%s", strings.Join(workFlow.ExplainStatus(), ","))
		t.Logf("finish check")
		println()
		return true, nil
	}
}
