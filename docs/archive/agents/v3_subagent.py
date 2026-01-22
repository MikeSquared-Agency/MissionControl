#!/usr/bin/env python3
"""Agent with subagent spawning (~450 lines). Isolated child agents for complex tasks."""

import os
import subprocess
import sys
import anthropic

client = anthropic.Anthropic()

# In-memory todo list
todos: list[dict] = []

# Track subagents
subagents: list[dict] = []

SYSTEM_PROMPT = """You are a helpful coding assistant that can delegate tasks to subagents.

IMPORTANT WORKFLOW:
1. For complex tasks, break them down using todo_add
2. For isolated subtasks, spawn a subagent using the 'task' tool
3. Subagents have their own context - give them clear, complete instructions
4. Review subagent results and integrate their work

When to use subagents:
- Tasks that can be done independently
- When you want isolated context (won't pollute your main conversation)
- Parallel-style work (research, then implement)

Tools:
- bash: Run shell commands
- read: Read file contents
- write: Create/overwrite files
- edit: Find and replace in files
- todo_add/todo_update/todo_list: Track your tasks
- task: Spawn a subagent for isolated work

Be strategic about delegation. You're the orchestrator."""

SUBAGENT_SYSTEM = """You are a focused coding assistant working on a specific task.
You have access to bash, read, write, and edit tools.
Complete your assigned task efficiently and report back.
Do not ask questions - make reasonable decisions and proceed."""

# Tools for main agent (includes task)
main_tools = [
    {
        "name": "bash",
        "description": "Run a bash command and return the output",
        "input_schema": {
            "type": "object",
            "properties": {
                "command": {"type": "string", "description": "The bash command to run"}
            },
            "required": ["command"],
        },
    },
    {
        "name": "read",
        "description": "Read the contents of a file",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Path to the file to read"}
            },
            "required": ["path"],
        },
    },
    {
        "name": "write",
        "description": "Write content to a file (creates or overwrites)",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Path to the file to write"},
                "content": {"type": "string", "description": "Content to write to the file"},
            },
            "required": ["path", "content"],
        },
    },
    {
        "name": "edit",
        "description": "Edit a file by replacing old_string with new_string. The old_string must match exactly.",
        "input_schema": {
            "type": "object",
            "properties": {
                "path": {"type": "string", "description": "Path to the file to edit"},
                "old_string": {"type": "string", "description": "The exact string to find and replace"},
                "new_string": {"type": "string", "description": "The string to replace it with"},
            },
            "required": ["path", "old_string", "new_string"],
        },
    },
    {
        "name": "todo_add",
        "description": "Add a new task to the todo list",
        "input_schema": {
            "type": "object",
            "properties": {
                "task": {"type": "string", "description": "Description of the task"}
            },
            "required": ["task"],
        },
    },
    {
        "name": "todo_update",
        "description": "Update the status of a task",
        "input_schema": {
            "type": "object",
            "properties": {
                "index": {"type": "integer", "description": "Task index (0-based)"},
                "status": {
                    "type": "string",
                    "enum": ["pending", "in_progress", "done"],
                    "description": "New status for the task"
                },
            },
            "required": ["index", "status"],
        },
    },
    {
        "name": "todo_list",
        "description": "List all tasks with their status",
        "input_schema": {
            "type": "object",
            "properties": {},
        },
    },
    {
        "name": "task",
        "description": "Spawn a subagent to handle an isolated task. The subagent has its own context and tools (bash, read, write, edit). Use for independent subtasks.",
        "input_schema": {
            "type": "object",
            "properties": {
                "description": {
                    "type": "string",
                    "description": "Clear, complete description of what the subagent should do"
                },
            },
            "required": ["description"],
        },
    },
]

# Tools for subagent (no task spawning - prevents infinite recursion)
subagent_tools = [t for t in main_tools if t["name"] not in ["task", "todo_add", "todo_update", "todo_list"]]


def execute_basic(tool_name: str, tool_input: dict) -> str:
    """Execute basic tools (shared by main and subagent)."""

    if tool_name == "bash":
        try:
            result = subprocess.run(
                tool_input["command"],
                shell=True,
                capture_output=True,
                text=True,
                timeout=120,
            )
            output = result.stdout + result.stderr
            return output if output else "(no output)"
        except subprocess.TimeoutExpired:
            return "Error: Command timed out after 120 seconds"
        except Exception as e:
            return f"Error: {e}"

    elif tool_name == "read":
        path = tool_input["path"]
        try:
            with open(path, "r") as f:
                content = f.read()
            return content if content else "(empty file)"
        except FileNotFoundError:
            return f"Error: File not found: {path}"
        except Exception as e:
            return f"Error reading file: {e}"

    elif tool_name == "write":
        path = tool_input["path"]
        content = tool_input["content"]
        try:
            os.makedirs(os.path.dirname(path) or ".", exist_ok=True)
            with open(path, "w") as f:
                f.write(content)
            return f"Successfully wrote {len(content)} bytes to {path}"
        except Exception as e:
            return f"Error writing file: {e}"

    elif tool_name == "edit":
        path = tool_input["path"]
        old_string = tool_input["old_string"]
        new_string = tool_input["new_string"]
        try:
            with open(path, "r") as f:
                content = f.read()

            if old_string not in content:
                return f"Error: old_string not found in {path}"

            count = content.count(old_string)
            if count > 1:
                return f"Error: old_string found {count} times. Make it more specific."

            new_content = content.replace(old_string, new_string)
            with open(path, "w") as f:
                f.write(new_content)
            return f"Successfully edited {path}"
        except FileNotFoundError:
            return f"Error: File not found: {path}"
        except Exception as e:
            return f"Error editing file: {e}"

    return None  # Signal that tool wasn't handled


def run_subagent(task_description: str) -> str:
    """Run a subagent with isolated context."""
    subagent_id = len(subagents)
    subagents.append({"task": task_description, "status": "running", "turns": 0})

    print(f"\n  [Subagent {subagent_id}] Starting: {task_description[:60]}...")

    messages = [{"role": "user", "content": task_description}]
    max_turns = 20  # Prevent runaway subagents

    for turn in range(max_turns):
        subagents[subagent_id]["turns"] = turn + 1

        response = client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=4096,
            system=SUBAGENT_SYSTEM,
            tools=subagent_tools,
            messages=messages,
        )

        # Process response
        assistant_content = []
        tool_results = []

        for block in response.content:
            assistant_content.append(block)

            if block.type == "tool_use":
                result = execute_basic(block.name, block.input)
                if result is None:
                    result = f"Unknown tool: {block.name}"

                # Brief logging
                if block.name == "bash":
                    print(f"  [Subagent {subagent_id}] $ {block.input.get('command', '')[:50]}")
                else:
                    print(f"  [Subagent {subagent_id}] [{block.name}]")

                tool_results.append({
                    "type": "tool_result",
                    "tool_use_id": block.id,
                    "content": result,
                })

        messages.append({"role": "assistant", "content": assistant_content})

        if response.stop_reason != "tool_use":
            subagents[subagent_id]["status"] = "done"
            final_text = "".join(b.text for b in response.content if hasattr(b, "text"))
            print(f"  [Subagent {subagent_id}] Done in {turn + 1} turns")
            return final_text

        messages.append({"role": "user", "content": tool_results})

    subagents[subagent_id]["status"] = "timeout"
    return f"Subagent reached max turns ({max_turns}). Partial work may be completed."


def execute(tool_name: str, tool_input: dict) -> str:
    """Execute a tool (main agent version)."""
    global todos

    # Try basic tools first
    result = execute_basic(tool_name, tool_input)
    if result is not None:
        return result

    # Todo tools
    if tool_name == "todo_add":
        task = tool_input["task"]
        todos.append({"task": task, "status": "pending"})
        return f"Added task {len(todos) - 1}: {task}"

    elif tool_name == "todo_update":
        index = tool_input["index"]
        status = tool_input["status"]
        if index < 0 or index >= len(todos):
            return f"Error: Invalid index {index}. Valid range: 0-{len(todos) - 1}"
        todos[index]["status"] = status
        return f"Updated task {index} to {status}"

    elif tool_name == "todo_list":
        if not todos:
            return "No tasks yet."
        lines = []
        status_icons = {"pending": "[ ]", "in_progress": "[~]", "done": "[x]"}
        for i, todo in enumerate(todos):
            icon = status_icons.get(todo["status"], "[ ]")
            lines.append(f"{i}. {icon} {todo['task']}")
        return "\n".join(lines)

    # Task tool (spawn subagent)
    elif tool_name == "task":
        description = tool_input["description"]
        return run_subagent(description)

    return f"Unknown tool: {tool_name}"


def format_tool_call(name: str, input: dict) -> str:
    """Format a tool call for display."""
    if name == "bash":
        return f"$ {input.get('command', '')}"
    elif name == "read":
        return f"[read] {input.get('path', '')}"
    elif name == "write":
        return f"[write] {input.get('path', '')} ({len(input.get('content', ''))} bytes)"
    elif name == "edit":
        return f"[edit] {input.get('path', '')}"
    elif name == "todo_add":
        return f"[todo+] {input.get('task', '')}"
    elif name == "todo_update":
        return f"[todo] #{input.get('index', '?')} -> {input.get('status', '?')}"
    elif name == "todo_list":
        return "[todo] listing..."
    elif name == "task":
        desc = input.get('description', '')[:50]
        return f"[task] Spawning subagent: {desc}..."
    return f"[{name}]"


def print_status():
    """Print current status."""
    parts = []
    if todos:
        done = sum(1 for t in todos if t["status"] == "done")
        parts.append(f"Todos: {done}/{len(todos)}")
    if subagents:
        running = sum(1 for s in subagents if s["status"] == "running")
        parts.append(f"Subagents: {len(subagents)} ({running} running)")
    if parts:
        print(f"  {' | '.join(parts)}")


def agent(task: str) -> str:
    """Run the main agent loop."""
    messages = [{"role": "user", "content": task}]
    turn = 0

    while True:
        turn += 1
        print(f"\n[Turn {turn}]")
        print_status()

        response = client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=4096,
            system=SYSTEM_PROMPT,
            tools=main_tools,
            messages=messages,
        )

        # Process response
        assistant_content = []
        tool_results = []

        for block in response.content:
            assistant_content.append(block)

            if hasattr(block, "text"):
                print(block.text)

            elif block.type == "tool_use":
                print(format_tool_call(block.name, block.input))
                result = execute(block.name, block.input)

                # Print result (truncated)
                display = result[:500] + "..." if len(result) > 500 else result
                if block.name != "task":  # Task already prints its own output
                    print(display)

                tool_results.append({
                    "type": "tool_result",
                    "tool_use_id": block.id,
                    "content": result,
                })

        messages.append({"role": "assistant", "content": assistant_content})

        if response.stop_reason != "tool_use":
            return "".join(b.text for b in response.content if hasattr(b, "text"))

        messages.append({"role": "user", "content": tool_results})


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python v3_subagent.py <task>")
        print('Example: python v3_subagent.py "build a todo app with CLI interface"')
        sys.exit(1)

    task = sys.argv[1]
    print(f"Task: {task}")
    print("=" * 60)
    result = agent(task)
    print("\n" + "=" * 60)
    print("[Done]")

    # Final summary
    if todos:
        print("\nTodos:")
        print(execute("todo_list", {}))
    if subagents:
        print(f"\nSubagents spawned: {len(subagents)}")
        for i, s in enumerate(subagents):
            print(f"  {i}. [{s['status']}] {s['task'][:50]}... ({s['turns']} turns)")
