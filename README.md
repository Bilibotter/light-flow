# LightFlow

**[English](README.md),  [中文](README.cn.md)**

LightFlow is a declarative task orchestration framework.

Users only need to focus on the timing of task execution, while the framework automatically handles the orchestration of tasks.

- **Declarative** : Focus on execution timing using functional programming.
- **[Mergeable Processes](./merge.md)**: Seamlessly integrate registered processes into the current workflow being constructed.
- **[Multilevel Callbacks](./callback.md)**: Support callbacks at various levels, allowing flexible inclusion of callback logic.
- **[Multilevel Configuration](./config.md)**: Each level can be configured with priorities for smaller-level configurations.
- **[Unique Context Mechanism](./context.md)**: Connect step contexts with direct or indirect dependencies and isolate step contexts without dependencies.
- **Maximum Execution Coverage**: Even if a step fails, other steps that do not depend on it will continue to execute.

### Installation

```
go get github.com/Bilibotter/light-flow
```

### Usage

```go
import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"strings"
)

func Step1(ctx flow.StepCtx) (result interface{}, err error) {
	// Context can get value set by workflow input
	if input, exists := ctx.Get("flow-input"); exists {
		fmt.Printf("%s get workflow input, value = '%s'\n", ctx.ContextName(), input.(string))
	}
	ctx.Set("step1-key", 1)
	result = "Step1 Result"
	return
}

func Step2(ctx flow.StepCtx) (result interface{}, err error) {
	if value, exists := ctx.Get("step1-key"); exists {
		fmt.Printf("%s get Step1 key, value = '%d'\n", ctx.ContextName(), value.(int))
	}
	// Context can get related step's execute result by step name
	if step1Result, exists := ctx.GetResult("Step1"); exists {
		fmt.Printf("%s get Step1 result, value = `%s`\n", ctx.ContextName(), step1Result.(string))
	}
	return
}

func AfterStepCallback(step *flow.Step) (keepOn bool, err error) {
	if step.Exceptions() != nil {
		fmt.Printf("%s executed faield %s occur\n", step.Name, strings.Join(step.Exceptions(), ","))
	}
	return true, nil
}

func init() {
	workflow := flow.RegisterFlow("WorkFlow")
	process := workflow.Process("Process")
	process.Step(Step1)
	// Step2 depends on Step1
	process.Step(Step2, Step1)
	// Use the callback function to handle errors during step execution
	workflow.AfterStep(false, AfterStepCallback)
}

func main() {
	// Complete the workflow with initial input values
	flow.DoneFlow("WorkFlow", map[string]interface{}{"flow-input": "Hello world!"})
}
```

