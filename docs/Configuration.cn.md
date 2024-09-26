### Configuration

#### 概述

多级配置机制使用户能够在不同层级（`Default`、`Flow`、`Process` 和 `Step`）灵活定义任务的配置项。各层级遵循优先级递增的规则，确保最终配置值的正确性和有效性。

---

#### 配置层级

多级配置的层级从全局默认设置开始，到具体的 `Step` 层级，优先级逐层递增。配置项的优先级顺序如下：

1. **Default（全局默认配置）**：框架的全局默认配置，适用于所有任务和步骤，优先级最低。如果未在其他层级配置该项，将使用默认值。

2. **Flow（任务流配置）**：配置适用于整个 `Flow` 任务流，覆盖全局 `Default` 配置项的值，适合为整个任务流设定统一的配置。

3. **Process（流程配置）**：`Process` 配置针对具体流程生效，优先级高于 `Flow`，覆盖 `Flow` 和 `Default` 中的配置值，适用于细化流程中的配置。

4. **Step（步骤配置）**：配置优先级最高，专门为特定 `Step` 定义。如果在 `Step` 层级上设置配置项，它将覆盖其他所有层级的配置。

---

#### 配置项生效规则

框架根据配置项的优先级选择最终生效的配置。每个 `Step` 都会遵循以下规则，逐级检查配置项：

- **Step 层级**：首先检查 `Step` 是否设置了配置项。如果设置了，优先使用 `Step` 配置。
- **Process 层级**：如果 `Step` 层级未配置该项，系统将检查对应的 `Process` 是否配置了该项。
- **Flow 层级**：如果 `Process` 层级也未配置该项，则检查 `Flow` 任务流的配置。
- **Default 层级**：如果在 `Flow` 层级也没有找到该配置项，系统将使用全局 `Default` 配置。

这种多级配置机制允许用户根据需求，在不同层级灵活覆盖和定制配置。

---

#### 可配置配置项

- **EnableRecover**：启用断点恢复，默认禁用。
- **ProcessTimeout**：设置流程的过期时间。
- **StepTimeout**：设置步骤的过期时间。
- **StepRetry**：步骤失败后的最大重试次数。

**示例**

```go
config := flow.DefaultConfig()
config.EnableRecover() // Enable recovery
config.ProcessTimeout(3 * time.Hour) // Set process timeout
config.StepTimeout(5 * time.Minute) // Set step timeout
config.StepRetry(4) // Set max retries for steps
```

---

#### 示例：StepTimeout 配置

假设有一个配置项 `StepTimeout`，用于设置任务的超时时间。你可以在 `Default`、`Flow`、`Process` 和 `Step` 层级分别进行配置。

1. **Default 层级**：设置默认的超时时间，适用于所有任务步骤。

   ```go
   flow.DefaultConfig().StepTimeout(10 * time.Minute) // Set default step timeout
   ```

2. **Flow 层级**：在 `Flow` 层级为整个任务流设置步骤默认超时时间。

   ```go
   wf := flow.RegisterFlow("WorkFlow")
   wf.StepTimeout(5 * time.Minute) // Set step timeout for the flow
   ```

3. **Process 层级**：为某个 `Process` 单独定义 Step 超时时间，覆盖 `Flow` 层级的配置。

   ```go
   process := wf.Process("Process")
   process.StepTimeout(1 * time.Minute) // Set step timeout for the process
   ```

4. **Step 层级**：为具体的 `Step` 定义超时，优先级最高，覆盖所有其他层级的配置。

   ```go
   step := process.CustomStep(Step1, "Step1")
   step.StepTimeout(1 * time.Minute) // Set step timeout for this specific step
   ```

在这个例子中，如果在 `Step` 层级设置了 `StepTimeout` 为 30 秒，那么该步骤执行时的超时会使用 `Step` 配置。如果没有设置，则会依次查找 `Process`、`Flow` 和 `Default` 层级的配置。

---

#### 使用场景

多级配置非常适用于以下场景：

1. **全局控制与局部覆盖**：用户可以为整个任务流设置默认配置（如超时时间、重试次数等），并在特定的 `Process` 或 `Step` 中进行局部覆盖，满足不同步骤的特定需求。

2. **灵活的层级控制**：配置项可以根据任务的复杂度或场景需要，灵活地在各个层级进行调整，避免冗余配置，提升配置管理的效率。

3. **简化复杂任务流的配置管理**：对于包含多个 `Process` 和 `Step` 的复杂任务流，多级配置可以减少重复定义，提供一个清晰的配置继承和覆盖机制，确保任务流的执行行为符合预期。

