# 多级配置

WorkFlow使用了多级配置，用户可以灵活地在各个层级使用不同的配置，实现定制化的任务处理流程。

- **四个配置层级**：从大到小依次为Default、Workflow、Process、Step。
- **配置优先级**：更小层级配置优先，Step > Process > Workflow > default。
- **NotUseDefault**：每个层级都可以单独设置NotUseDefault选项，表示当前层级及其包含的层级不使用默认配置。

**示例代码**

```go
import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
)

func RetryStep(ctx flow.StepCtx) (any, error) {
	retryCount := 1

	if retry, exist := ctx.Get("retry"); exist {
		retryCount = retry.(int) + 1
	}

	fmt.Printf("[Step - %s] Retry Count: %d\n", ctx.ContextName(), retryCount)
	ctx.Set("retry", retryCount)

	return nil, fmt.Errorf("[Step - %s] error occurred", ctx.ContextName())
}

func init() {
	config := flow.CreateDefaultConfig()
	config.StepRetry = 4

	workflow := flow.RegisterFlow("WorkFlow")
	workflow.StepRetry = 3

	process := workflow.Process("Process")
	process.StepRetry = 2

	step := process.Step(RetryStep)
	step.Retry(1)
}

func main() {
	// Expected Output:
	// [Step - RetryStep] Retry Count: 1
	flow.DoneFlow("WorkFlow", nil)
}
```

