# Multi-level Callback

- **Execution Order**: Callbacks are executed in the order of WorkFlow -> Process -> Step.
- **Callback Hierarchy**: Arranged and executed in the order of Default -> WorkFlow -> Process.
- **Callback Setting**: Higher levels can set callbacks for lower levels, allowing flexible addition and management of callback logic during task execution. For example, Default, WorkFlow, and Process can all set callbacks for Step.
- **Callback Types**: Essential callbacks failing will interrupt the current level task, while non-essential callbacks failing do not affect subsequent tasks.
- **Chained Calls**: Callbacks can interrupt subsequent callbacks at the same level, but only within the current level. If a Default's Step callback is interrupted, WorkFlow and Process's Step callbacks will still be executed.
- **Modifiers**: By using OnlyFor and NotFor, callbacks can be limited to execute or not execute at specific steps. Using When and Exclude can restrict callbacks to execute or not execute based on specific states. Setting NotUseDefault will exclude the use of default callbacks.

**Example**

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