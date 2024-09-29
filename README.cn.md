# LightFlow：基于函数式编程的任务编排框架

LightFlow 是一个基于 Go 语言的任务编排框架，旨在简化复杂任务流的设计和管理。它通过函数式编程直接在代码中定义任务流，**将重点从全局任务编排转移到任务执行时机**，让开发者更专注于任务执行的时机控制，而无需维护复杂的配置文件或规则语言。

**LightFlow的优点**

1. **简化任务流管理**：不依赖外部规则语言或编排文件，所有任务依赖通过函数式编程定义在代码中，避免频繁切换上下文。
2. **聚焦执行时机**：开发者只需指定任务的依赖步骤，框架自动处理任务的顺序，减少了全局依赖管理的复杂性。
3. **提高维护性和可扩展性**：即使任务流的规模不断扩大，确定任务执行时机仍然是一个相对独立且基本的问题。

## 核心特性

- [**隔离性上下文**](./docs/Context.cn.md)：各`Step`通过隔离性上下文联系，仅访问相关上下文数据，避免全局混乱。
- [**编排基于执行时机**](./docs/Arrange.cn.md)：通过函数式编程定义任务流，灵活指定任务的执行时机，实现高效的任务管理与调度。
- [**流程可合并**](./docs/Merge.cn.md)：允许将已编排的流程合并到正在编排的流程中，优化流程管理。
- [**资源管理**](./docs/Resource.cn.md)：自动处理资源的附加、释放及断点恢复。
- [**断点恢复**](./docs/Recover.cn.md)：任务失败后，支持从失败点继续执行，避免重复运行。
- [**条件执行**](./docs/Condition.cn.md)：根据条件动态控制任务的执行与跳过。
- [**多级回调**](./docs/Callback.cn.md)：支持在多个层级设置回调，灵活管理任务状态。
- [**事件处理**](./docs/Event.cn.md)：处理任务执行阶段以外的错误，允许为每个阶段配置事件处理器。
- [**自定义持久化插件**](https://github.com/Bilibotter/light-flow-plugins/blob/main/README.cn.md)：用户可以自定义持久化插件，LightFlow不与任何 ORM 框架产生耦合，增强灵活性和可扩展性。

---

## 快速上手

###  安装

```sh
go get github.com/Bilibotter/light-flow/flow
```

### 示例代码

以下示例展示了如何使用 Light-Flow 进行简单的任务编排：

```go
package main

import (
	"fmt"
	"github.com/Bilibotter/light-flow/flow"
)

func First(step flow.Step) (any, error) {
	if input, exist := step.Get("input"); exist {
		fmt.Printf("[Step: %s] get input: %v\n", step.Name(), input)
	}
	step.Set("key", "value")
	return "result", nil
}

func Second(step flow.Step) (any, error) {
	if value, exist := step.Get("key"); exist {
		fmt.Printf("[Step: %s] get key: %v\n", step.Name(), value)
	}
	if result, exist := step.Result(step.Dependents()[0]); exist {
		fmt.Printf("[Step: %s] get result: %v\n", step.Name(), result)
	}
	return nil, nil
}

func ErrorStep(step flow.Step) (any, error) {
	if value, exist := step.Get("key"); exist {
		fmt.Printf("[Step: %s] get key: %v\n", step.Name(), value)
	} else {
		fmt.Printf("[Step: %s] cannot get key \n", step.Name())
	}
	return nil, fmt.Errorf("execute failed")
}

func ErrorHandler(step flow.Step) (bool, error) {
	if step.Has(flow.Failed) {
		fmt.Printf("[Step: %s] has failed\n", step.Name())
	} else {
		fmt.Printf("[Step: %s] success\n", step.Name())
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

### 输出示例

执行上述代码时，你将看到如下输出：

```shell
[Step: First] get input: Hello world
[Step: First] success
[Step: Second] get key: value
[Step: Second] get result: result
[Step: Second] success
[Step: ErrorStep] cannot get key 
[Step: ErrorStep] has failed
```

### 使用步骤

1. **定义步骤**：编写你的 `Step` 函数，使用 `step.Get` 和 `step.Set` 来与`Context`进行交互。[查看Context文档](./docs/Context.cn.md)
2. **创建流程**：在 `init` 函数中创建一个流程并添加`Step`。[查看编排文档](./docs/Arrange.cn.md)
3. **错误处理**：使用回调来处理各种异常情况。[查看回调文档](./docs/Callback.cn.md)
4. **启动执行**：调用 `flow.DoneFlow` 方法开始执行流程，并传入必要的输入数据

> 任务系统旨在尽最大努力执行所有任务，即便某个任务失败，只有依赖于该任务的后续任务会被放弃执行，同时链式取消相关任务。与失败任务无关的其他任务将继续运行。
>
> 这一设计理念支持断点恢复，确保系统在出现错误时仍能有效地执行未受影响的部分。

------

## 贡献与支持

如果你对 Light-Flow 有任何建议或问题，欢迎提交 issue 或 pull request。我们期待你的参与！
