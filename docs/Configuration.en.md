# Configuration

## Overview

The multi-level configuration mechanism allows users to flexibly define task configuration items at different levels (`Default`, `Flow`, `Process`, and `Step`). Each level follows an incrementing priority rule to ensure the correctness and effectiveness of the final configuration values.

---

## Configuration Levels

The levels of multi-level configuration start from the global default settings and progress to the specific `Step` level, with priority increasing at each level. The priority order of configuration items is as follows:

1. **Default**: The global default configuration for the framework, applicable to all tasks and steps, with the lowest priority. If an item is not configured at other levels, the default value will be used.

2. **Flow**: Configuration applicable to the entire `Flow` task, overriding the global `Default` values. This is suitable for setting uniform configurations for the whole task flow.

3. **Process**: `Process` configurations take effect for specific processes, with higher priority than `Flow`, overriding values in `Flow` and `Default`. This is used for fine-tuning configurations within the process.

4. **Step**: The highest priority configuration, specifically defined for a particular `Step`. If a configuration item is set at the `Step` level, it will override configurations at all other levels.

---

## Configuration Item Effectiveness Rules

The framework selects the final effective configuration based on the priority of configuration items. Each `Step` will follow these rules to check configuration items progressively:

- **Step Level**: First, check if the `Step` has a configuration item set. If so, the `Step` configuration will be used.
- **Process Level**: If the `Step` level does not have the item configured, the system will check if the corresponding `Process` has the item configured.
- **Flow Level**: If the `Process` level also does not have the item configured, it will check the configuration of the `Flow` task.
- **Default Level**: If the item is not found at the `Flow` level, the system will use the global `Default` configuration.

This multi-level configuration mechanism allows users to flexibly override and customize configurations at different levels based on their needs.

---

## Configurable Items

- **EnableRecover**: Enables recovery from checkpoints, disabled by default.
- **ProcessTimeout**: Sets the expiration time for the process.
- **StepTimeout**: Sets the expiration time for the step.
- **StepRetry**: Maximum number of retries after a step failure.

**Example**

```go
config := flow.DefaultConfig()
config.EnableRecover() // Enable recovery
config.ProcessTimeout(3 * time.Hour) // Set process timeout
config.StepTimeout(5 * time.Minute) // Set step timeout
config.StepRetry(4) // Set max retries for steps
```

---

## Example: StepTimeout Configuration

Assuming there is a configuration item `StepTimeout` used to set the task timeout. You can configure this at the `Default`, `Flow`, `Process`, and `Step` levels.

1. **Default Level**: Set the default timeout applicable to all task steps.

   ```go
   flow.DefaultConfig().StepTimeout(10 * time.Minute) // Set default step timeout
   ```

2. **Flow Level**: Set the default step timeout for the entire task flow at the `Flow` level.

   ```go
   wf := flow.RegisterFlow("WorkFlow")
   wf.StepTimeout(5 * time.Minute) // Set step timeout for the flow
   ```

3. **Process Level**: Define a separate step timeout for a specific `Process`, overriding the configuration at the `Flow` level.

   ```go
   process := wf.Process("Process")
   process.StepTimeout(1 * time.Minute) // Set step timeout for the process
   ```

4. **Step Level**: Define the timeout for a specific `Step`, which has the highest priority and overrides configurations from all other levels.

   ```go
   step := process.CustomStep(Step1, "Step1")
   step.StepTimeout(1 * time.Minute) // Set step timeout for this specific step
   ```

In this example, if the `Step` level sets `StepTimeout` to 30 seconds, that timeout will be applied when executing the step. If not set, the configuration will be checked in order from `Process`, `Flow`, and then `Default`.

---

## Use Cases

Multi-level configuration is particularly useful in the following scenarios:

1. **Global Control with Local Override**: Users can set default configurations (such as timeout and retry counts) for the entire task flow, with the ability to locally override them in specific `Process` or `Step` contexts to meet specific needs.

2. **Flexible Hierarchical Control**: Configuration items can be adjusted flexibly across various levels based on the complexity or requirements of the task, avoiding redundant configurations and improving configuration management efficiency.

3. **Simplifying Configuration Management for Complex Task Flows**: For complex task flows containing multiple `Process` and `Step` components, multi-level configuration reduces the need for repetitive definitions, providing a clear mechanism for configuration inheritance and overriding to ensure that the execution behavior of the task flow meets expectations.