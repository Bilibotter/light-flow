# light-flow

## Introduction

light-flow is a task arrange framework.

Designed to provide the most efficient orchestration and execution strategy for tasks with dependencies.

## Features

**Efficient Task Planning**: The framework allows you to define task dependencies, 
enabling the efficient planning and sequencing of steps based on their relationships

**Context Connect And Isolation**: Tasks can only access to the context of dependent tasks up to the root task.
Modifications to the context by the current task will not affect disconnected tasks.

**Rich Test Case**:  Test cases cover every public API and most scenarios.

**Minimal Code Requirement**: With this framework, you only need to write a minimal amount of code to define and execute your tasks. 

**Task Dependency Visualization**:The framework provides a draw plugin to visualize the dependencies between tasks. 

## Installation

Install the framework by running the following command:

`go get gitee.com/MetaphysicCoding/light-flow`

## Draw your flow

### Step 1: Define Step Functions

Define the step functions that will be executed in the workflow. Each step function should have a specific signature and return any results or errors.

```go
import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"time"
)

func Step1(ctx *flow.Context) (any, error) {
    // If we have previously set the 'name' in the global context or in the dependent steps, 
    // then we can retrieve the 'name'
	name, exist := ctx.Get("name")
	if exist {
		fmt.Printf("step1 get 'name' from ctx: %s \n", name.(string))
	}
	ctx.Set("age", 18)
	return "finish", nil
}

func Step2(ctx *flow.Context) (any, error) {
	age, exist := ctx.Get("age")
	if exist {
		fmt.Printf("step2 get 'age' from ctx: %d \n", age.(int))
	}
	result, exist := ctx.GetStepResult("Step1")
	if exist {
		fmt.Printf("step2 get result from step1: %s \n", result.(string))
	}
	return nil, nil
}
```

### Step 2: Create Work-Flow And Process

```go
workflow := flow.NewWorkflow[any](input)
config := flow.ProcessConfig{
	StepRetry:   3,
	StepTimeout: 30 * time.Minute,
}
// Process of workflow are parallel.
process := workflow.AddProcess("process1", &config)
workflow := flow.NewWorkflow[any](input)
config := flow.ProcessConfig{
    StepRetry:   3,
    StepTimeout: 30 * time.Minute,
}
```

### Step 3: Add Step And Define Dependencies

```go
// Add a step named "Step1" to process.
// Step 1 will use the function name as key.
// Although you can try AddStepWithAlias("Step1", Step1).
process.AddStep(Step1)
// Add a step named "Step2" to process and specify that Step2 depends on Step1.
// It is available to specify multiple dependent.
// AddStep(Step2, "Step1") is ok.
process.AddStep(Step2, Step1)
```

### Step 4: Run Work-Flow

```go
// Done will run work-flow and block until all processes are completed.
// Although you can use workflow.Flow(), it's a asynchronous method.
features := workflow.Done()
```

### Step 5: Get Execute Result By Features

```go
for name, feature := range features {
    if feature.Success() {
			// ExplainStatus compresses the status of all steps in the process.
			fmt.Printf("process[%s] result: %v \n", name, feature.ExplainStatus())
    } else {
        fmt.Printf("process[%s] occur error: %v \n", name, feature.ExplainStatus())
    }
}
```

### Complete Example

```go
import (
	"fmt"
	flow "gitee.com/MetaphysicCoding/light-flow"
	"time"
)

func Step1(ctx *flow.Context) (any, error) {
	name, exist := ctx.Get("name")
	if exist {
		fmt.Printf("step1 get 'name' from ctx: %s \n", name.(string))
	}
	ctx.Set("age", 18)
	return "finish", nil
}

func Step2(ctx *flow.Context) (any, error) {
	age, exist := ctx.Get("age")
	if exist {
		fmt.Printf("step2 get 'age' from ctx: %d \n", age.(int))
	}
	result, exist := ctx.GetStepResult("Step1")
	if exist {
		fmt.Printf("step2 get result from step1: %s \n", result.(string))
	}
	return nil, nil
}

func MyWorkFlow(input map[string]any) {
	workflow := flow.NewWorkflow[any](input)
	config := flow.ProcessConfig{
		StepRetry:   3,
		StepTimeout: 30 * time.Minute,
	}
	process := workflow.AddProcess("process1", &config)
	process.AddStep(Step1)
	process.AddStep(Step2, Step1)
	features := workflow.Done()
	for name, feature := range features {
		if feature.Success() {
			fmt.Printf("process[%s] result: %v \n", name, feature.ExplainStatus())
		} else {
			fmt.Printf("process[%s] occur error: %v \n", name, feature.ExplainStatus())
		}
	}
}

func main() {
	MyWorkFlow(map[string]any{"name": "foo"})
}
```

## Features Introduction



### Context Connect And Isolation

We define dependencies like flow code.

```go
workflow := flow.NewWorkflow[any](nil)
procedure := workflow.AddProcess("process", &conf)
procedure.AddStep(TaskA)
procedure.AddStep(TaskB, TaskA)
procedure.AddStep(TaskC, TaskA)
procedure.AddStep(TaskD, TaskB, TaskC)
procedure.AddStep(TaskE, TaskC)
```

The dependency relationship is shown in the figure

![Relation](./process.png)

TaskD can access the context of TaskB，TaskC，TaskA.

TaskD can access the context of TaskC，TaskA, but TaskD can't access the context of TaskB.

```
Note:  

Key first match in its own context, then matched in parents context, finally matched in global contxt. -->

You can use AddPriority to break the order 
```

