# LightFlow: A Task Orchestration Framework Based on Functional Programming

[English](./README.md) | [中文](./README.cn.md)

LightFlow is a task orchestration framework built in Go, designed to make managing complex task flows easier. It lets you define task flows directly in code using functional programming, **focusing on execution timing** instead of dealing with complicated configuration files or rules.

**Benefits of LightFlow**

1. **Simplified Task Management**: You don’t need external rule languages or orchestration files; all task dependencies are defined in your code, reducing context switching.
2. **Focus on Execution Timing**: Just specify the tasks that need to be completed first; the framework automatically manages the order of execution, making dependency management simpler.
3. **Better Maintainability and Scalability**: Even as task flows grow, determining when tasks should run remains straightforward.

## Core Features

- [**Isolated Contexts**](./docs/Context.en.md): Each `Step` is linked through isolated contexts, allowing access only to relevant data and preventing global confusion.
- [**Orchestration Based on Timing**](./docs/Arrange.en.md): Define task flows with functional programming, flexibly specifying when tasks should run for efficient management.
- [**Mergeable Flows**](./docs/Merge.en.md): Combine existing flows into new ones, making process management smoother.
- [**Resource Management**](./docs/Resource.en.md): Automatically handles resource allocation, release, and recovery from checkpoints.
- [**Checkpoint Recovery**](./docs/Recover.en.md): Supports resuming tasks from where they failed, avoiding repeated runs.
- [**Conditional Execution**](./docs/Condition.en.md): Control task execution based on specific conditions.
- [**Multi-Level Callbacks**](./docs/Callback.en.md): Set callbacks at various levels to manage task status flexibly.
- [**Event Handling**](./docs/Event.en.md): Handle errors outside of task execution, allowing for event handlers to be set for each stage.
- [**Custom Persistence Plugins**](https://github.com/Bilibotter/light-flow-plugins/blob/main/README.md): Users can create custom persistence plugins, and LightFlow is not coupled with any ORM framework.

---

## Quick Start

### Installation

```sh
go get github.com/Bilibotter/light-flow/flow
```

### Example

Here’s a simple example showing how to use LightFlow for task orchestration:

```go
package main

import (
	"fmt"
	"github.com/Bilibotter/light-flow/flow"
)

func First(step flow.Step) (any, error) {
	if input, exists := step.Get("input"); exists {
		fmt.Printf("[Step: %s] got input: %v\n", step.Name(), input)
	}
	step.Set("key", "value")
	return "result", nil
}

func Second(step flow.Step) (any, error) {
	if value, exists := step.Get("key"); exists {
		fmt.Printf("[Step: %s] got key: %v\n", step.Name(), value)
	}
	if result, exists := step.Result(step.Dependents()[0]); exists {
		fmt.Printf("[Step: %s] got result: %v\n", step.Name(), result)
	}
	return nil, nil
}

func ErrorStep(step flow.Step) (any, error) {
	if value, exists := step.Get("key"); exists {
		fmt.Printf("[Step: %s] got key: %v\n", step.Name(), value)
	} else {
		fmt.Printf("[Step: %s] cannot get key \n", step.Name())
	}
	return nil, fmt.Errorf("execution failed")
}

func ErrorHandler(step flow.Step) (bool, error) {
	if step.Has(flow.Failed) {
		fmt.Printf("[Step: %s] has failed\n", step.Name())
	} else {
		fmt.Printf("[Step: %s] succeeded\n", step.Name())
	}
	return true, nil
}

func init() {
	process := flow.FlowWithProcess("Example")
	process.Follow(First, Second)
	process.Follow(ErrorStep)
	process.AfterStep(true, ErrorHandler)
}

func main() {
	flow.DoneFlow("Example", map[string]any{"input": "Hello world"})
}
```

### Output

When you run this code, you will see:

```shell
[Step: First] got input: Hello world
[Step: First] succeeded
[Step: Second] got key: value
[Step: Second] got result: result
[Step: Second] succeeded
[Step: ErrorStep] cannot get key 
[Step: ErrorStep] has failed
```

### Steps to Use

1. **Define Steps**: Write your `Step` functions, using `step.Get` and `step.Set` to interact with the context. [See Context Documentation](./docs/Context.en.md)
2. **Create Flows**: Set up a flow and add steps in the `init` function. [See Orchestration Documentation](./docs/Arrange.en.md)
3. **Error Handling**: Use callbacks to manage different error cases. [See Callback Documentation](./docs/Callback.en.md)
4. **Start Execution**: Call the `flow.DoneFlow` method to run the flow, passing in the required input data.

> LightFlow is designed to try to run all tasks, even if one fails. Only the tasks that depend on a failed task will be skipped, while unrelated tasks will continue running. 
>
> This approach supports checkpoint recovery, ensuring that the system can still execute parts that are unaffected by errors.

---

## Contribution and Support

If you have any suggestions or questions about LightFlow, please feel free to submit an issue or pull request. We welcome your input!
