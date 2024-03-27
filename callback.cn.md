# 多级回调

- **执行顺序**：回调顺序为 WorkFlow -> Process -> Step。

- **回调层级**： 按照Default -> WorkFlow -> Process 的顺序排列与执行。
- **回调设置**：高层级可以为低层级设置回调，使得任务执行过程中能够灵活地添加和管理回调逻辑。例如Default、WorkFlow、Process都可以设置Step的回调。
- **回调类型**：必要回调失败会中断当前层级任务，非必要回调失败不影响后续任务。
- **链式调用**：回调可中断同级后续回调，但只限于当前层级。若Default的Step回调中断，则仍会执行WorkFlow和Process的Step回调。
- **限定词**：通过OnlyFor和NotFor可以限定回调在特定的步骤执行或不执行，通过When和Exclude可以限定回调在特定的状态执行或不执行。当设置NotUaseDefault时不会使用默认回调。

**示例代码**

```go
import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
)

func SimpleStep(ctx flow.StepCtx) (any, error) {
	fmt.Printf("[Step - %s] finish\n", ctx.ContextName())
	return nil, nil
}

func GenerateCallback(name string) func(*flow.Step) (keepOn bool, err error) {
	return func(step *flow.Step) (keepOn bool, err error) {
		fmt.Printf("[Step - %s] invoke callback set by [%s]\n", step.ContextName(), name)
		return true, nil
	}
}

func init() {
	config := flow.CreateDefaultConfig()
	config.AfterStep(false, GenerateCallback("Default-Config")).When(flow.Success)

	workflow := flow.RegisterFlow("WorkFlow")
	workflow.AfterStep(false, GenerateCallback("WorkFlow")).OnlyFor("SimpleStep")

	process := workflow.Process("Process")
	process.AfterStep(false, GenerateCallback("Process"))

	process.Step(SimpleStep)
}

func main() {
	// Expected Output:
	// [Step - SimpleStep] finish
	// [Step - SimpleStep] invoke callback set by [Default-Config]
	// [Step - SimpleStep] invoke callback set by [WorkFlow]
	// [Step - SimpleStep] invoke callback set by [Process]
	flow.DoneFlow("WorkFlow", nil)
}
```

