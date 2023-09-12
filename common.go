package light_flow

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	InternalPrefix = "~"
	WorkflowCtx    = "~workflow~"
	ProcessCtx     = "~process~"
)

const (
	NoUse = int64(-1)
	End   = int64(0)
	Start = int64(1)
)

const (
	Pending    = int64(0)
	Running    = int64(0b1)
	Pause      = int64(0b1 << 1)
	Success    = int64(0b1 << 15)
	NormalMask = int64(0b1<<16 - 1)
	Cancel     = int64(0b1 << 16)
	Timeout    = int64(0b1 << 17)
	Panic      = int64(0b11 << 18)
	Failed     = int64(0b1 << 31)
	// abnormal produce status will cause the cancellation of unexecuted steps during execution
	// abnormal step status will cause the cancellation of unexecuted steps that depend on it
	AbnormalMask = NormalMask << 16
)

var (
	normal = map[int64]string{
		Pending: "Pending",
		Running: "Running",
		Pause:   "Pause",
		Success: "Success",
	}

	abnormal = map[int64]string{
		Cancel:  "Cancel",
		Timeout: "Timeout",
		Panic:   "Panic",
		Failed:  "Failed",
	}
)

type Context struct {
	scope         string
	scopeContexts map[string]*Context
	parents       []*Context
	table         sync.Map // current node's context
	priority      map[string]any
}

type Feature struct {
	running *sync.WaitGroup
	status  *int64
	finish  *int64
}

func AppendStatus(addr *int64, update int64) (change bool) {
	for status := atomic.LoadInt64(addr); status&update != update; {
		if atomic.CompareAndSwapInt64(addr, status, status|update) {
			return true
		}
		status = atomic.LoadInt64(addr)
	}
	return false
}

func IsStatusNormal(status int64) bool {
	if status&AbnormalMask == 0 {
		return true
	}
	return false
}

func ExplainStatus(status int64) []string {
	contains := make([]string, 0)

	for flag, name := range abnormal {
		if flag&status == flag {
			contains = append(contains, name)
		}
	}
	if len(contains) > 0 {
		return contains
	}

	if status&Success == Success {
		return []string{"Success"}
	}

	for flag, name := range normal {
		if flag&status == flag {
			contains = append(contains, name)
		}
	}

	return contains
}

func GetIndex(value any) string {
	var index string
	switch value.(type) {
	case func(ctx *Context) (any, error):
		index = GetFuncName(value)
	case string:
		index = value.(string)
	default:
		panic("index's value must be func(ctx *Context) (any, error) or string")
	}
	return index
}

func (ctx *Context) Set(key string, value any) {
	ctx.table.Store(key, value)
}

func (ctx *Context) Get(key string) (any, bool) {
	if used, exist := ctx.getCtxByPriority(key); exist {
		return used, true
	}

	defer func() {
	}()
	if used, exist := ctx.table.Load(key); exist {
		return used, true
	}

	for _, parent := range ctx.parents {
		used, exist := parent.Get(key)
		if exist {
			return used, exist
		}
	}

	return nil, false
}

func (ctx *Context) GetStepResult(name string) (any, bool) {
	key := InternalPrefix + name
	return ctx.Get(key)
}

func (ctx *Context) checkPriority() {
	for key := range ctx.priority {
		ctx.getCtxByPriority(key)
	}
}

func (ctx *Context) getCtxByPriority(key string) (*Context, bool) {
	value, exist := ctx.priority[key]
	if !exist {
		return nil, false
	}
	index := GetIndex(value)
	used, exist := ctx.scopeContexts[index]
	if !exist {
		// If priority's value is next node, it will cause infinite loop
		// So priority's value must exist.
		panic(fmt.Sprintf("can't find context named %s", index))
	}
	return used, true
}

func (f *Feature) WaitToDone() {
	f.running.Wait()
	// wait goroutine to change status
	for finish := atomic.LoadInt64(f.finish); finish == 0; finish = atomic.LoadInt64(f.finish) {
	}
}

func (f *Feature) IsSuccess() bool {
	return *f.status&Success == Success && IsStatusNormal(atomic.LoadInt64(f.status))
}

func (f *Feature) GetStatusExplain() []string {
	return ExplainStatus(atomic.LoadInt64(f.status))
}
