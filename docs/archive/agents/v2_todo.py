#!/usr/bin/env python3
"""Agent with todo tracking (~300 lines). Explicit planning capability."""

import os
import subprocess
import sys
import json
import anthropic

client = anthropic.Anthropic()

# In-memory todo list
todos: list[dict] = []

SYSTEM_PROMPT = """You are a helpful coding assistant with the ability to track tasks.

IMPORTANT: For any non-trivial task, ALWAYS start by creating a todo list to plan your approach.
This helps you stay organized and ensures you don't miss steps.

Workflow:
1. Break down the task into steps using todo_add
2. Work through each step, marking them in_progress then done
3. Use todo_list to check your progress

Tools:
- bash: Run shell commands
- read: Read file contents
- write: Create/overwrite files
- edit: Find and replace in files
- todo_add: Add a task to your list
- todo_update: Update task status (pending/in_progress/done)
- todo_list: View all tasks

Be methodical. Plan first, then execute."""

tools = [
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
]


def execute(tool_name: str, tool_input: dict) -> str:
    """Execute a tool and return the result."""
    global todos

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

    elif tool_name == "todo_add":
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
        return "[todo] listing tasks..."
    return f"[{name}]"


def print_todos():
    """Print current todo status."""
    if not todos:
        return
    done = sum(1 for t in todos if t["status"] == "done")
    total = len(todos)
    in_progress = sum(1 for t in todos if t["status"] == "in_progress")
    print(f"  Progress: {done}/{total} done, {in_progress} in progress")


def agent(task: str) -> str:
    """Run the agent loop until completion."""
    messages = [{"role": "user", "content": task}]
    turn = 0

    while True:
        turn += 1
        print(f"\n[Turn {turn}]")
        print_todos()

        response = client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=4096,
            system=SYSTEM_PROMPT,
            tools=tools,
            messages=messages,
        )

        # Process response content
        assistant_content = []
        tool_results = []

        for block in response.content:
            assistant_content.append(block)

            if hasattr(block, "text"):
                print(block.text)

            elif block.type == "tool_use":
                print(format_tool_call(block.name, block.input))
                result = execute(block.name, block.input)

                # Print result (truncated if long)
                display = result[:500] + "..." if len(result) > 500 else result
                print(display)

                tool_results.append({
                    "type": "tool_result",
                    "tool_use_id": block.id,
                    "content": result,
                })

        messages.append({"role": "assistant", "content": assistant_content})

        # If no tool use, we're done
        if response.stop_reason != "tool_use":
            return "".join(b.text for b in response.content if hasattr(b, "text"))

        messages.append({"role": "user", "content": tool_results})


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python v2_todo.py <task>")
        print('Example: python v2_todo.py "build a calculator CLI app"')
        sys.exit(1)

    task = sys.argv[1]
    print(f"Task: {task}")
    result = agent(task)
    print(f"\n[Done]")

    # Print final todo summary
    if todos:
        print("\nFinal todo status:")
        print(execute("todo_list", {}))
