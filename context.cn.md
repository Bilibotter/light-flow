### Context特性总结

任务编排框架的Context机制类似于OverlayFS和Git，用于实现键值的设置和传递。Workflow的输入、Process、Step之间通过Context进行连接，并实现信息共享。具体特性如下：

1. **键值设置机制**: 类似 OverlayFS 和 Git，Context 允许用户设置键值，并保留历史版本。
2. **层级连通**：Workflow的输入、Process、Step通过Context相互关联，实现不同层级之间的数据传递和共享。
3. **隔离性**：不同Process内的Step相互隔离，即使它们属于同一个Workflow。同属于一个Process但无直接或间接依赖关系的Step也相互独立。
4. **键值传递**：Workflow在Context中设置的键值可以被从属的Process和Step获取，而Process设置的键值也可被其从属的Step获取。
5. **依赖项传递**：Step可以获取直接依赖的Step或间接依赖的Step设置的键值，确保依赖项之间的信息传递。
6. **同名键处理**：若多个直接或间接依赖的Step设置了同名键，则获取的值为最后一次设置的键值。采用优先级机制指定某个键从指定Step的Context中获取，以解决同名键冲突。
7. **Set方法特性**：Context的Set方法新增键值对而不修改已有值，确保先前设置的键值稳定性。

**示例代码**

```go
import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
)

func BeforeProcessIncrement(info *flow.Process) (keepOn bool, err error) {
	if count, exist := info.Get("count"); exist {
		current := count.(int)
		fmt.Printf("[Before Process - %s] Initial Count: %d\n", info.ContextName(), current)
		info.Set("count", current+1)
	}
	return true, nil
}

func StepIncrement(ctx flow.StepCtx) (any, error) {
	if count, exist := ctx.Get("count"); exist {
		current := count.(int)
		fmt.Printf("[Step - %s] Incremented Count: %d\n", ctx.ContextName(), current)
		ctx.Set("count", current+1)
	}
	return nil, nil
}

func init() {
	workflow := flow.RegisterFlow("IsolatedTest")
	process := workflow.Process("IsolatedTest")
	process.BeforeProcess(false, BeforeProcessIncrement)
	process.NameStep(StepIncrement, "1.1").Next(StepIncrement, "1.2")
	process.NameStep(StepIncrement, "2.1").Next(StepIncrement, "2.2")
}

func main() {
	// Expected Output:
	// [Before Process - IsolatedTest] Initial Count: 1
	// [Step - 2.1] Incremented Count: 2
	// [Step - 2.2] Incremented Count: 3
	// [Step - 1.1] Incremented Count: 2
	// [Step - 1.2] Incremented Count: 3
	flow.DoneFlow("IsolatedTest", map[string]interface{}{"count": 1})
}
```

