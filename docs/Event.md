### Event

#### 1. 简介

在 Light-Flow中，事件处理器用于处理除执行阶段之外的各种错误和异常（`panic`）。你可以为每个阶段（如 `InCallback`、`InRecover`、`InSuspend` 等）配置事件处理链来管理这些错误。事件处理器支持灵活的配置，比如协程数控制、超时设置等，以确保任务流的稳定性。

---

#### 2. 事件级别

框架中的事件分为两种级别：

- **Error**：表示常规错误，事件的 `Error()` 方法返回对应的错误。
- **Panic**：表示程序运行中的 `panic`，`Panic()` 返回Panic对象，`StackTrace()` 方法帮助你获取 `panic` 的详细信息和堆栈。

通过这些事件级别，你可以根据不同的错误情况执行不同的处理逻辑。

---

#### 3. 如何处理事件

框架允许你为每个阶段（如 `InCallback`、`InRecover` 等）配置一个事件处理链。处理器会在该阶段捕获到错误或异常时执行对应的处理逻辑。

##### 示例：设置 `InCallback` 阶段的事件处理器

```go
// 创建事件处理器并配置错误和 panic 处理逻辑
eventProcessor := NewEventProcessor().
    OnError(func(event Event) {
        // 处理 Error 事件
        log.Printf("Error in %s: %s", event.Name(), event.Error())
    }).
    OnPanic(func(event Event) {
        // 处理 Panic 事件
        log.Printf("Panic in %s, StackTrace: %s", event.Name(), event.StackTrace())
    })

// 为 InCallback 阶段设置事件处理链
flow := NewFlow().
    SetEventProcessor(InCallback, eventProcessor)
```

在这个例子中，事件处理器会处理 `InCallback` 阶段的 `Error` 和 `Panic` 事件，并将错误信息记录到日志中。

---

#### 4. 获取事件的详细信息

每个事件都会包含丰富的上下文信息，帮助你理解事件发生的背景：

- **`DetailMap()`**：返回一个 `map[string]string`，包含事件的详细信息。
- **`Layer()`**：指出错误发生的层级，可能是 `Flow`、`Process` 或 `Step`。
- **`ID()` 和 `Name()`**：返回出错单元的唯一标识符和名称，便于你快速定位问题。
- **`EventID()`**：每个事件都有一个唯一的 `EventID`，用于跟踪和识别事件。

##### 示例：获取事件详细信息

```go
eventProcessor := NewEventProcessor().
    OnError(func(event Event) {
        details := event.DetailMap()
        layer := event.Layer()
        log.Printf("Error in %s (Layer: %s), Details: %v", event.Name(), layer, details)
    })
```

在此示例中，处理器不仅记录了错误，还从事件中提取了详细信息和层级，用于更精确地调试问题。

---

#### 5. 读取上下文数据

你可以使用事件对象的 `Get(key string)` 方法从上下文中获取键值对。注意，事件处理是异步的，因此**你只能读取上下文数据**，不能修改。

##### 示例：从上下文中读取数据

```go
eventProcessor := NewEventProcessor().
    OnError(func(event Event) {
        value := event.Get("some_key")
        log.Printf("Context value for key 'some_key': %v", value)
    })
```

---

#### 6. 配置事件处理器

你可以通过多种方式配置事件处理器，以控制事件处理的行为：

- **最大协程数**：处理器的协程数是动态调节的，但你可以通过设置最大协程数来防止资源耗尽。
- **超时设置**：每个事件处理链可以配置超时时间，确保处理器在规定时间内完成任务，避免长时间阻塞。
- **丢弃策略**：在处理大量事件时，你可以配置处理器丢弃低优先级的事件，防止积压。

##### 示例：配置事件处理器

```go
// 设置事件处理器的最大协程数、超时时间和日志记录
eventProcessor := NewEventProcessor().
    SetMaxGoroutines(10).
    SetTimeout(5 * time.Second).
    OnError(func(event Event) {
        log.Printf("Error in %s: %s", event.Name(), event.Error())
    })

// 应用于 InSuspend 阶段
flow := NewFlow().
    SetEventProcessor(InSuspend, eventProcessor)
```

在这个示例中，事件处理器配置了最大协程数为 10，并设置了 5 秒的超时限制。如果事件处理超时，会自动停止。

---

#### 7. 完整示例

以下是一个完整的示例，展示了如何为不同阶段设置事件处理器并配置协程和超时：

```go
// 为多个阶段设置事件处理器
eventProcessor := NewEventProcessor().
    SetMaxGoroutines(20).           // 设置最大协程数
    SetTimeout(10 * time.Second).    // 设置超时时间
    OnError(func(event Event) {
        log.Printf("Error occurred: %s", event.Error())
    }).
    OnPanic(func(event Event) {
        log.Printf("Panic occurred in %s, Stack: %s", event.Name(), event.StackTrace())
    })

// 应用于 InRecover 阶段
flow := NewFlow().
    SetEventProcessor(InRecover, eventProcessor).
    SetEventProcessor(InCallback, eventProcessor)

// 执行任务流
flow.Execute()
```

---

#### 8. 总结

通过事件处理器，框架能够灵活地处理任务流中的各种错误和异常。你可以为不同阶段定制处理链，通过配置协程、超时和上下文访问来控制事件处理行为。