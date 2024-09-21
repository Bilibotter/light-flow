# 多级回调机制特性

## 0. 使用建议 

- 在绝大部分情况下，只需使用Default回调配合 `For` 与 `NotFor` 即可满足要求。Flow 和 Process 级别的回调配置主要适用于测试平台的任务编排场景，特别是需要大量验证的场景。

## 1. 分层执行

- 回调可以在 `Flow`、`Process` 和 `Step` 这三个层级上定义前置（Before）和后置（After）回调。
- 回调按照定义的来源层级顺序执行：
  1. **Default**：全局回调，可定义 Flow、Process 和 Step 回调。
  2. **Flow**：特定 Flow 的回调，可定义 Flow、Process 和 Step 回调。
  3. **Process**：特定 Process 的回调，可定义 Process 和 Step 回调。
  
  以Step的回调执行顺序举例。
  
  ```mermaid
  graph LR
      A[Default BeforeStep] --> B[Flow BeforeStep]
      B --> C[Process BeforeStep]
      C --> D[Step Execution]
      D --> E[Default AfterStep]
      E --> F[Flow AfterStep]
      F --> G[Process AfterStep]
  ```

## 2. 回调的状态限定

- 使用 `When` 或 `Exclude` 方法限定回调仅在特定状态下或不在特定状态下执行，例如 `flow.Failed` 状态。
- 使用 `For` 或 `NotFor` 方法限定回调仅在特定单元中或不在特定单元中执行。
- 使用 `If` 方法使回调仅在某个函数返回 `true` 的情况下执行。
- 通过状态和名称的限定，可以更精准地控制回调的执行时机，确保只在需要时才执行相应的逻辑。

## 3. 必要与非必要回调

- **非必要回调**：在注册回调时，`must` 参数传入 `false` 表示回调为非必要。如果失败，流程会继续，不会中断。非必要回调用于非关键操作，如日志记录或通知。

- **必要回调**：在注册回调时，`must` 参数传入 `true` 表示回调为必要。如果回调标记为必要，则其失败会导致当前单元（Flow 或 Process）直接退出，或者相关的后续 Step 被跳过。

  必要回调可以视作单元的一部分，只有当单元及其必要回调都执行成功时，该单元才被视为执行成功。
  
  前置回调执行顺序及必要前置回调失败的影响如下。
  
  ```mermaid
  flowchart TB
  	H[Cancel Execute Step]
  	A -->|must-callback failed| H
      A[Default BeforeStep] -->|success| B[Flow BeforeStep]
      B -->|must-callback failed| H
      B -->|success| C[Process BeforeStep]
      C -->|must-callback failed| H
      C --> |success|D[Execute Step]   
  ```
  
  后置回调执行顺序及必要后回调失败的影响如下。
  
  ```mermaid
  flowchart TB
  	H[Step Failed]
  	I[Execute Step] --> A
  	A -->|must-callback failed| H
      A[Default AfterStep] -->|success| B[Flow AfterStep]
      B -->|must-callback failed| H
      B -->|success| C[Process AfterStep]
      C -->|must-callback failed| H
  ```
  
  

## 4. `DisableDefaultCallback` 控制

- `DisableDefaultCallback` 可以在 `Flow` 或 `Process` 级别使用，禁用默认回调。禁用后，只有特定的 Flow 或 Process 不会执行默认回调，其他 Flow 和 Process 依然会执行默认回调。
- 当回调失败时，日志会记录失败回调的层级（Default、Process 或 Flow）、回调链中的顺序位置及类型（Flow、Process、Step），帮助调试问题。

## 5. 链式调用与中断机制

- 回调机制是链式调用的，允许通过返回 `false` 中断后续回调，但只能中断当前层级的回调执行。
  - 例如，当 Default 设置的回调中断时，Flow 和 Process 层级的回调仍会继续执行。
- 这种设计保证了不同层级回调之间的独立性，同时提供了针对特定场景的中断控制。



