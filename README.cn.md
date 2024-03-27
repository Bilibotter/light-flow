# LightFlow

LightFlow是一个免编排的任务编排框架。

用户只需将注意力放在任务的执行时机上，而任务的编排由框架自动完成。

- **免编排**：专注于执行时机，使用函数式编程。
- **[可合并流程](./merge.cn.md)**：无缝地将已注册的流程合并到正在构建的流程中。
- **[多级回调](./callback.cn.md)**：各层级支持回调，灵活加入回调逻辑。
- **[多级配置](./config.cn.md)**：每个层级都可以设置配置，较小层级配置具有优先级。
- **[独特上下文机制](./context.cn.md)**：连接具有直接或间接依赖关系的步骤上下文，隔离没有依赖关系的步骤上下文。
- **尽最大可能执行**：即使某步骤失败，不依赖该步骤的其他步骤仍会继续执行。

### 安装

`go get github.com/Bilibotter/light-flow`

### 使用

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

