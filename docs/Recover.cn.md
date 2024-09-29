## Recover

#### 概述

断点恢复功能允许任务在执行失败时，从上次中断的地方继续执行。恢复过程涉及任务的主要逻辑以及 `Context` 和 `Resource` 的状态恢复，以确保任务在恢复后能够与最初执行时保持一致。

[Context文档](./Context.cn.md)|[Resource文档](./Resource.cn.md)

---

#### 启用断点恢复

默认情况下，断点恢复功能处于禁用状态。要启用该功能，需在 `Flow` 对象上调用 `EnableRecover()` 方法，或通过默认配置启用断点恢复。

```go
// Enable Recover in Flow
wf := flow.RegisterFlow("Recover")
wf.EnableRecover()

// Enable Recover in Default 
flow.DefaultConfig().EnableRecover()
```

启用后，当任务执行失败时，框架会生成当前 `Context` 和 `Resource` 的快照，并通过序列化方式保存。任务恢复要求用户对 `Context` 和 `Resource` 中的非基本类型进行注册（使用 `RegisterType`）。

```go
// Register a custom type using generics
flow.RegisterType[YourStruct]()
```

---

#### 恢复机制

- **快照生成**: 当可恢复的任务执行失败时，框架会生成 `Context` 和 `Resource` 的快照，并通过 gob 序列化后，调用用户提供的持久化接口进行保存。

- **任务恢复**: 任务失败后，框架会挂起任务并记录失败点。在恢复时，框架会重新加载保存的快照，从失败的 `Step` 或前置 `Callback` 继续执行（断点恢复）。恢复过程中，后置 `Callback` 也会被重新执行（回放）。

- **幂等性保证**: 在恢复过程中，`Step` 和后置回调可能会被多次执行，因此必须确保它们的幂等性。为避免重复执行，可以在后置回调中调用 `Exclude(Recovering)`，跳过恢复时的重复执行。建议始终保证后置回调的幂等性，以确保任务恢复时的行为与最初执行时一致。

---

#### 资源恢复

Light-Flow通过持久化 `Context` 实现框架内部的挂起，同时使用资源来抽象外部系统，以配合断点恢复的需求。

在任务执行和恢复过程中，资源的状态通过 `Suspend` 和 `Recover` 方法进行管理。有关资源管理的更多信息，请参阅 [资源文档](./Resource.cn.md)。

---

#### 持久化与敏感数据处理

框架提供灵活的持久化接口，允许用户自定义 `Context` 和 `Resource` 的保存与加载方式，并选择合适的数据库或存储方案。

**示例：使用 GORM 插件实现持久化**

以下示例展示了如何使用 GORM 插件实现自定义持久化：

```go
import plugins "github.com/Bilibotter/light-flow-plugins/orm"

// Open a database connection
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
// Inject the suspend plugin for persistence
err := plugins.NewSuspendPlugin(db).InjectSuspend()
```

------

对于敏感数据，建议在 `Suspend` 时清空这些数据，并在 `Recover` 时重新生成，以避免将敏感信息直接存储到数据库中。如果无法通过资源管理器处理敏感数据，可以使用 Light-Flow 的加密功能。

**示例：加密敏感数据**

使用 `AES256` 对 `pwd` 和 `password` 这两个键进行加密，并将 `"secret"` 作为秘钥：

```go
// Set AES256 encryptor for sensitive keys
flow.SetEncryptor(flow.NewAES256Encryptor([]byte("secret"), "pwd", "password"))
```

---

#### 执行恢复

有两种恢复方式可供选择：

1. 使用 `DoneFlow` 产生的 `FinishedFlow` 进行恢复。
2. 使用 `RecoverWorkFlow` 方法来恢复对应 `FlowId` 的 `WorkFlow`。

**示例1：使用 FinishedWorkFlow 进行恢复**

```go
// Create a FinishedFlow instance and recover from it
ff := flow.DoneFlow("Recover", nil)
// Call Recover method to restore the workflow state
ff.Recover()
```

**示例2：使用 FlowId 进行恢复**

```go
// Restore the workflow using its FlowId
flow.RecoverFlow("xxxxxxxxxxx")
```

**示例3：模拟步骤失败然后进行恢复**

```go
import (
	"fmt"
	plugins "github.com/Bilibotter/light-flow-plugins/orm"
	"github.com/Bilibotter/light-flow/flow"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type YourStruct struct {
	ID   int
	Name string
}

// Step1 function simulates a task step that may fail
func Step1(step flow.Step) (any, error) {
	fmt.Printf("[Step: %s] start\n", step.Name())
    // If recovering, simulate success; otherwise, simulate failure.
	if step.Has(flow.Recovering) {
		fmt.Printf("[Step: %s] success\n", step.Name())
		return nil, nil
	}
	step.Set("key", YourStruct{ID: 1, Name: "John"})
	fmt.Printf("[Step: %s] failed\n", step.Name())
	return nil, fmt.Errorf("something went wrong")
}

// Step2 function retrieves data set by Step1 and completes successfully.
func Step2(step flow.Step) (any, error) {
	fmt.Printf("[Step: %s] start\n", step.Name())
	value, _ := step.Get("key")
	fmt.Printf("[Step: %s] get value: %v\n", step.Name(), value)
	fmt.Printf("[Step: %s] success\n", step.Name())
	return nil, nil
}

func init() {
	dsn := "xxx"
	db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	plugins.NewSuspendPlugin(db).InjectSuspend()
    
	flow.RegisterType[YourStruct]()
    
	wf := flow.RegisterFlow("Recover")
	wf.EnableRecover()
    
	wf.Process("Recover").Follow(Step1, Step2)
}

func main() {
	ff := flow.DoneFlow("Recover", nil)
	fmt.Printf("======Recover======\n")
	ff.Recover()
}
```

---

#### 多次恢复

如果任务恢复失败，框架会再次挂起任务，等待下一个恢复机会。这一机制确保任务不会因为一次恢复失败而永久中断。

需要注意的是，只有执行失败的 `WorkFlow` 才能进行恢复。