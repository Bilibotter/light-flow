# light-flow

## Introduction

light-flow is a task arrange framework.

Designed to provide the most efficient orchestration and execution strategy for tasks with dependencies

## Features

**Efficient Task Planning**: The framework allows you to define task dependencies, enabling the efficient planning and sequencing of steps based on their relationships

**Context Connect And Isolation**: Tasks can only access to the context of dependent tasks up to the root task.Modifications to the context by the current task will not affect disconnected tasks.

**Rich Test Case**:  Test cases cover every public API and most scenarios.

**Minimal Code Requirement**: With this framework, you only need to write a minimal amount of code to define and execute your tasks. It abstracts away the complexities of task orchestration, allowing you to focus on the logic specific to each task.

**Task Dependency Visualization**:The framework provides a draw plugin to visualize the dependencies between tasks. This graphical representation allows you to easily understand the flow and relationships between tasks.