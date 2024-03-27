### Process Merge Feature

- WorkFlow supports merging pre-designed process sequences into the current ongoing process to simplify task management and enhance reusability.

  **Feature Summary:**

    1. **Process Merging Feature**: Allows users to merge a completed designed process into the current workflow, enhancing process reusability and flexibility.
    2. **Step Handling Logic**: During merging, new steps are added to the target workflow in the order of the source process, and existing steps' dependency relationships are automatically updated.
    3. **Circular Dependency Detection**: The framework detects circular dependencies during merging. If detected, it will panic and notify which two steps are involved in the cycle.

**Example Code:**

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

