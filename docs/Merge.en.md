1. ### Merge 

   #### Overview

   The `Merge` feature of `Light-Flow` allows users to reuse registered `Processes` and seamlessly integrate them into the current task flow. During the merging process, execution steps, configurations, and conditions are automatically adjusted to ensure that the merged flow operates correctly. If a circular dependency is detected in the execution order, the system will trigger a `Panic` and indicate the location of the cycle.

   [Condition Execution Documentation](./Condition.en.md) [Configuration Documentation](./Configuration.en.md)

   ---

   #### Introduction to Merge Functionality

   - **Step Merging**: When a registered `Process` is merged, the framework automatically adjusts the order of execution steps. If a step does not exist in the current `Process`, the system will automatically add it to ensure the integrity of the flow.

     **Example: How to Merge Two Processes**:

     The execution flow of registered `Process A` is as follows:

     ```mermaid
     %%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#333', 'lineColor': '#333', 'textColor': 'black' } } }%%
     flowchart LR;
         A["Step 1"]-->B["Step 2"];
         B --> C["Step 3"];
         C --> D["Step 5"];
     linkStyle default stroke:#888888
     classDef default fill:#98FF98,stroke:#333,stroke-width:2px;
     ```

     The execution flow of the current `Process B` is as follows:

     ```mermaid
     %%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#333', 'lineColor': '#333', 'textColor': 'black' } } }%%
     flowchart LR;
         A["Step 2"]-->B["Step 3"];
         C["Step 4"] --> D["Step 5"]
     linkStyle default stroke:#888888
     classDef default fill:#98FF98,stroke:#333,stroke-width:2px;
     ```

     After merging `Process A` into `Process B`, the execution flow is as follows:

     ```mermaid
     %%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#333', 'lineColor': '#333', 'textColor': 'black' } } }%%
     flowchart LR;
         A["Step 1"]-->B["Step 2"];
         B --> C["Step 3"];
         C --> D["Step 5"];
         E["Step 4"] --> D
     linkStyle default stroke:#888888
     classDef default fill:#98FF98,stroke:#333,stroke-width:2px;
     ```

   **Example**:

   ```go
   import (
   	"fmt"
   	"github.com/Bilibotter/light-flow/flow"
   	"strings"
   )
   
   func Step1(step flow.Step) (any, error) {
   	fmt.Printf("Executing [Step: %s], Dependents= [%s]\n", step.Name(), strings.Join(step.Dependents(), ", "))
   	return nil, nil
   }
   
   // Similar functions for Step2, Step3, Step4, Step5
   
   func init() {
   	merged := flow.FlowWithProcess("A")
   	merged.Follow(Step1, Step2, Step3, Step5)
   
   	target := flow.FlowWithProcess("B")
   	target.Merge("A") // Merge Process A into Process B
   	target.Follow(Step2, Step3)
   	target.Follow(Step4, Step5);
   }
   
   func main() {
   	flow.DoneFlow("B", nil) // Execute the merged process
   }
   ```

   - **Configuration Merging**: During the merging process, step-related configurations and conditions are also automatically merged. For configurations like `StepTimeout`, the framework provides a priority inheritance mechanism to ensure a reasonable configuration override logic.

   **Example: Merging Configurations and Conditions**:

   ```go
   merged := flow.FlowWithProcess("A")
   step := merged.CustomStep(Step2, "Step1")
   step.EQ("temperature", 30)
   step.Restrict(map[string]any{"Key": Step2})
   step.StepTimeout(5 * time.Minute) // Set timeout for Step1
   
   target := flow.FlowWithProcess("B")
   target.Merge("A") // Merge Process A into Process B
   target.Follow(Step1);
   ```

   ---

   #### Configuration Priority

   During merging, the priority of configurations follows this order:

   1. **Current Step's Settings** (highest priority)
   2. **Current Process's Settings**
   3. **Current Flow's Settings**
   4. **Merged Step's Settings**
   5. **Global Default Configuration** (lowest priority)

   This priority mechanism allows users to configure at more granular levels, providing flexible control over task behavior while retaining merged process configurations.

   ---

   #### Condition Merging Mechanism

   During merging, the framework retains each step's conditions. These conditions define when a step will execute under specific circumstances, ensuring that execution logic remains unaffected. [See Condition Documentation](./Condition.en.md)

   ---

   #### Cycle Detection Mechanism

   The framework automatically checks for cycles between Steps. If a circular dependency is detected, the system will immediately trigger a `Panic` and indicate which two Steps form the cycle, helping users quickly locate and resolve issues.

   **Example**:

   If there is a cyclic dependency like `Step1 -> Step2 -> Step3 -> Step4 -> Step1`, the system will panic and indicate that there is a cycle between `Step4 -> Step1`.

   ```mermaid
   %%{init: {'theme': 'neutral', 'themeVariables': { 'primaryColor': '#333', 'lineColor': '#333', 'textColor': 'black' } } }%%
   flowchart LR;
       A["Step 1"]-->B["Step 2"];
       B --> C["Step 3"];
       C --> D["Step 4"]
       D --> A
   %% Mark Step1 and Step4 in red to indicate the cycle
   	linkStyle default stroke:#888888
   	style B fill:#98FF98,stroke:#333,stroke-width:2px;
   	style C fill:#98FF98,stroke:#333,stroke-width:2px;
       style A fill:#ff6666,stroke:#333,stroke-width:2px;
       style D fill:#ff6666,stroke:#333,stroke-width:2px;
   ```

   ---

   #### Steps for Performing Merge

   1. **Register Process**: Before merging, ensure that the Process to be merged has been registered.
   2. **Execute Merge**: Use the `Merge` interface to combine the registered Process into the currently orchestrated Process.
   3. **Adjust Execution Order**: The system will automatically adjust the execution order and check for any missing steps to ensure completeness.
   4. **Cycle Detection**: If a cycle is detected, the system will trigger a `Panic` and provide detailed error information.