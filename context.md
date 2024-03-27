### Context Feature Summary

The Context mechanism in this task orchestration framework is similar to OverlayFS and Git, allowing for key-value setting and propagation. The input of Workflow, Process, and Step are connected through Context, facilitating information sharing.

## Key Features:

1. **Key-Value Setting Mechanism**: Similar to OverlayFS and Git, the Context enables users to set key-value pairs while retaining historical versions.
2. **Hierarchical Connectivity**: The input of Workflow, Process, and Step are interconnected through Context, enabling data transfer and sharing across different levels.
3. **Isolation**: Steps within different Processes are isolated, even if they belong to the same Workflow. Steps within the same Process but without direct or indirect dependencies are also independent.
4. **Key-Value Propagation**: Key-values set in Workflow's Context can be accessed by its associated Processes and Steps. Similarly, key-values set in a Process can be retrieved by its associated Steps.
5. **Dependency Propagation**: Steps can access key-values set by directly dependent Steps or indirectly dependent Steps, ensuring information transfer between dependencies.
6. **Handling of Same Name Keys**: If multiple directly or indirectly dependent Steps set keys with the same name, the value retrieved will be from the last set key-value. A priority mechanism can specify retrieving a key from a specific Step's Context to resolve conflicts of same name keys.
7. **Set Method Feature**: The Set method of Context adds key-value pairs without modifying existing values, ensuring the stability of previously set key-values.

By leveraging these features, users can efficiently manage and propagate key-value pairs within the workflow, enhancing the flexibility and robustness of the orchestration process.

**Example**

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

