### Process合并特性

WorkFlow支持将已经设计好执行顺序的流程合并到当前正在进行的流程中，以简化任务管理并提高可复用性。

**特性总结：**

1. **Process合并功能**：允许用户将已完成设计的流程合并到当前流程中，提高流程的复用性和灵活性。
2. **Step处理逻辑**：合并时，新的步骤按照源流程的顺序被添加到目标流程中，已存在步骤的依赖关系会被自动更新。
3. **环形依赖检测**：合并时，框架会检测是否出现环形依赖，若检测到则会panic，并告知是哪两个Step之间存在环。

**示例代码：**

```go
import (
	"fmt"
	flow "github.com/Bilibotter/light-flow"
)

const (
	SourceProcess = "SourceProcess"
	TargetProcess = "TargetProcess"
)

func Step1(ctx flow.StepCtx) (any, error) {
	fmt.Println("Executing Step1")
	return nil, nil
}

func Step2(ctx flow.StepCtx) (any, error) {
	fmt.Println("Executing Step2")
	return nil, nil
}

func Step3(ctx flow.StepCtx) (any, error) {
	fmt.Println("Executing Step3")
	return nil, nil
}

func init() {
	source := flow.RegisterFlow("SourceFlow")
	process := source.Process(SourceProcess)
	process.Step(Step1).Next(Step2)

	target := flow.RegisterFlow("TargetFlow")
	process = target.Process(TargetProcess)
	process.Step(Step2).Next(Step3)
	process.Merge("SourceProcess")
}

func main() {
	// Expected Output:
	// Executing Step1
	// Executing Step2
	// Executing Step3
	flow.DoneFlow("TargetFlow", nil)
}
```

