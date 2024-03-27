# Multi-level Configuration

WorkFlow utilizes a multi-level configuration approach, enabling users to flexibly apply different configurations at various levels to achieve a customized task processing flow.

- **Four Configuration Levels**: The levels, in descending order, are Default, Workflow, Process, and Step.
- **Configuration Priority**: Smaller level configurations take precedence over larger level configurations. Specifically, Step configurations have higher priority than Process configurations, Process configurations have higher priority than Workflow configurations, and so on.
- **NotUseDefault**: Each level can individually set the NotUseDefault option, indicating that the current level and its contained levels should not utilize the default configuration. This empowers users to have granular control over configuration inheritance and customization.

**Example**

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

