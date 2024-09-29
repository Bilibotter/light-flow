### Event

#### 1. Introduction

In Light-Flow, event handlers are essential tools for managing errors and exceptions. They handle various errors and exceptions that occur outside of execution stage (such as `panic`). You can configure separate event handling chains for each execution phase (like `InCallback`, `InRecover`, `InSuspend`, etc.) to effectively capture and manage these errors. This flexible configuration supports controlling the number of goroutines and setting timeouts, enhancing the stability and reliability of task flows.

---

#### 2. Event Levels

Events in the framework are divided into two main levels:

- **Error**: Represents general errors. You can get specific error information using the event's `Error()` method.

- **Panic**: Indicates a `panic` that occurs during program execution. Use the `Panic()` method to get the Panic object and the `StackTrace()` method for detailed panic information and stack tracing.

By distinguishing between these two event levels, you can take appropriate actions based on different types of errors.

---

#### 3. Event Stage

Each phase has its own independent event handler. Here are the configurable event stage:

- **InCallback**: Handles Errors and Panics that occur during callback processing. [See Callback Documentation](./Callback.en.md)

- **InPersist**: Handles Errors and Panics that occur during the persistence of Flow, Process, and Step execution records.

- **InSuspend**: Handles Errors and Panics that occur when a WorkFlow is suspended. [See Recover Documentation](./Recover.en.md)

- **InRecover**: Handles Errors and Panics that occur when a WorkFlow is recovered. [See Recover Documentation](./Recover.en.md)

- **InResource**: Handles Errors and Panics that occur during resource management. [See Resource Documentation](./Resource.en.md)

---

#### 4. How to Handle Events

You can configure an event handling chain for each phase (like `InCallback`, `InRecover`, etc.). The event handler will execute the corresponding logic when it captures an error or exception. You can also set up discard logic, where new events will be discarded if the event queue is full and reaches its maximum capacity.

##### Example: Setting Up an Event Handler for the InCallback Phase

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

In this example, the event handler is specifically designed to capture and log `Error` and `Panic` events during the `InCallback` phase.

---

#### 5. Get Event Details

Each event contains rich contextual information that helps understand the background of an error. Here are some important methods:

- **`DetailMap()`**: Returns a `map[string]string` containing detailed information about the event.

- **`Layer()`**: Indicates the layer where the error occurred, such as `Flow`, `Process`, or `Step`.

- **`ID()` and `Name()`**: Return the ID and name of the unit where the error occurred, allowing for quick problem identification.

- **`EventID()`**: Each event has a unique identifier used to track and identify it.

- **`FlowID()` and `FlowName()`**: Get the ID and name of the WorkFlow.

- **`ProcessID()` and `ProcessName()`**: Get the ID and name of the Process. If the unit where the error occurred is at the `Flow` level, it will return an empty string.

---

#### 6. Reading Context Data

You can get key-value pairs from context using the method `Get(key string)` on the event object. Note that event handling occurs asynchronously, so you can only read context data without modifying it.

##### Example: Reading Data from Context

```go
func GetCtx(event flow.FlexEvent) (keepOn bool) {
	value, _ := event.Get("key")
	fmt.Printf("Get key: %s\n", value)
	return true
}
```

---

#### 7. Configuring Event Handlers

You can configure event handlers in various ways to control their behavior:

- **Maximum Goroutines**: Dynamically adjust the number of goroutines to prevent resource exhaustion. You can set a maximum number of goroutines to limit concurrent processing capacity.

- **Timeout Settings**: Configure timeout durations for each event handling chain to ensure tasks are completed within a specified time frame, avoiding prolonged blocking.

##### Example: Configuring Event Handlers

```go
// Configure event handler settings for optimal performance
flow.EventHandler().
    MaxHandler(8).              // Set maximum goroutines to handle events concurrently
    Capacity(24).               // Set capacity of the event queue to manage load effectively
    EventTimeoutSec(300)       // Set timeout limit for each event handler to prevent long waits
```

In this example, the event handler is configured to allow a maximum of 8 goroutines, with a queue capacity of 24, while setting a maximum time limit of 300 seconds for each event handling process.