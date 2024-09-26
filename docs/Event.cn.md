### Event

#### 1. 简介

在 Light-Flow 中，事件处理器是管理错误和异常的重要工具。事件处理器用于管理执行阶段之外的各种错误和异常（例如 `panic`）。您可以为每个执行阶段（如 `InCallback`、`InRecover`、`InSuspend` 等）配置独立的事件处理链，以便有效地捕获和处理这些错误。这种灵活的配置支持协程数量控制和超时设置，从而增强任务流的稳定性和可靠性。

---

#### 2. 事件级别

框架中的事件分为两种主要级别：

- **Error**：表示常规错误。您可以通过事件的 `Error()` 方法获取具体的错误信息。
  
- **Panic**：表示程序运行中的 `panic`。使用 `Panic()` 方法获取 Panic 对象，使用 `StackTrace()` 方法获取详细的 `panic` 信息及堆栈跟踪。

通过区分这两种事件级别，您可以根据不同类型的错误采取相应的处理措施。

---

#### 3. 事件阶段

每个阶段都有独立的事件处理器，以下是可配置的事件阶段：

- **InCallback**: 处理回调过程中产生的 Error 和 Panic。[Callback文档](./Callback.cn.md)
  
- **InPersist**: 处理 Flow、Process 和 Step 执行记录持久化时产生的 Error 和 Panic。
  
- **InSuspend**: 处理 WorkFlow 挂起时产生的 Error 和 Panic。[Recover文档](./Recover.cn.md)
  
- **InRecover**: 处理 WorkFlow 恢复时产生的 Error 和 Panic。[Recover文档](./Recover.cn.md)
  
- **InResource**: 处理资源管理过程中产生的 Error 和 Panic。[Resource文档](./Resource.cn.md)

---

#### 4. 如何处理事件

您可以为每个阶段（如 `InCallback`、`InRecover` 等）配置事件处理链。事件处理器会在捕获到错误或异常时执行相应的逻辑。您还可以设置丢弃逻辑，当事件队列已满且达到最大容量时，将丢弃新到达的事件。

##### 示例：设置 `InCallback` 阶段的事件处理器

```go
import (
	"fmt"
	"github.com/Bilibotter/light-flow/flow"
)

func CallbackErrorHandler(event flow.FlexEvent) (keepOn bool) {
	if event.Level() != flow.ErrorLevel {
		return true
	}
	// May output like `[Step: step1] execute fail | error: something went wrong`
	fmt.Printf("[%s: %s] execute fail | error: %s\n", event.Layer(), event.Name(), event.Error())
	fmt.Printf("Detail: %s\n", event.DetailsMap())
	return true
}

func CallbackPanicHandler(event flow.FlexEvent) (keepOn bool) {
	if event.Level() != flow.PanicLevel {
		return true
	}
	fmt.Printf("[%s: %s] execute fail | panic: %s\n", event.Layer(), event.Name(), event.Panic())
	fmt.Printf("Panic trace: %s\n", event.StackTrace())
	return true
}

func DiscardEvent(event flow.FlexEvent) (keepOn bool) {
	return true
}

func init() {
	flow.EventHandler().
		Handle(flow.InCallback, CallbackErrorHandler, CallbackPanicHandler).
    	Discard(flow.InCallback, DiscardEvent).
		DisableLog() // If the handler is already logging errors, you can disable the framework's logging of such events.
}
```

在这个示例中，事件处理器专门用于捕获和记录 `InCallback` 阶段的 `Error` 和 `Panic` 事件。

---

#### 5. 获取事件详细信息

每个事件都包含丰富的上下文信息，有助于理解错误发生的背景。以下是一些重要的方法：

- **`DetailMap()`**：返回一个包含事件详细信息的 `map[string]string`。
  
- **`Layer()`**：指示错误发生的层级，例如 `Flow`、`Process` 或 `Step`。
  
- **`ID()` 和 `Name()`**：返回出错单元的 ID 和名称，以便快速定位问题。
  
- **`EventID()`**：每个事件都有一个唯一标识符，用于跟踪和识别该事件。
  
- **`FlowID()` 和 `FlowName()`**：获取 WorkFlow 的 ID 和名称。
  
- **`ProcessID()` 和 `ProcessName()`**：获取 Process 的 ID 和名称。如果出错单元层级为 `Flow`，则返回空字符串。

---

#### 6. 读取上下文数据

您可以通过事件对象的方法 `Get(key string)` 从上下文中获取键值对。请注意，事件处理是异步进行的，因此您只能读取上下文数据，而不能进行修改。

##### 示例：从上下文中读取数据

```go
func GetCtx(event flow.FlexEvent) (keepOn bool) {
	value, _ := event.Get("key")
	fmt.Printf("Get key: %s\n", value)
	return true
}
```

---

#### 7. 配置事件处理器

您可以通过多种方式配置事件处理器，以控制其行为：

- **最大协程数**：动态调节协程数以防止资源耗尽。您可以设置最大协程数来限制并发处理能力。
  
- **超时设置**：为每个事件处理链配置超时时间，以确保在规定时间内完成任务，避免长时间阻塞。

##### 示例：配置事件处理器

```go
// Configure event handler settings for optimal performance
flow.EventHandler().
    MaxHandler(8).              // Set maximum goroutines to handle events concurrently
    Capacity(24).               // Set the capacity of the event queue to manage load effectively
    EventTimeoutSec(300)       // Set timeout limit for each event handler to prevent long waits
```

在这个示例中，事件处理器被配置为最大允许 8 个协程，并且设置了队列容量为 24，同时规定了每个事件处理最多可用 300 秒。

