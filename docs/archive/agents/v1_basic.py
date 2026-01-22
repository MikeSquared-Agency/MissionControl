#!/usr/bin/env python3
"""Basic agent with file tools (~200 lines). A complete coding agent."""

import os
import subprocess
import sys
import anthropic

client = anthropic.Anthropic()

SYSTEM_PROMPT = """You are a helpful coding assistant. You have access to tools for running bash commands and manipulating files.

When editing files:
- Use 'read' first to see current contents
- Use 'edit' for surgical changes (find and replace)
- Use 'write' only for new files or complete rewrites

Be concise. Execute tasks efficiently."""

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
]


def execute(tool_name: str, tool_input: dict) -> str:
    """Execute a tool and return the result."""

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
            # Create parent directories if needed
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

            # Check for multiple occurrences
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
    return f"[{name}]"


def agent(task: str) -> str:
    """Run the agent loop until completion."""
    messages = [{"role": "user", "content": task}]
    turn = 0

    while True:
        turn += 1
        print(f"\n[Turn {turn}]")

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
        print("Usage: python v1_basic.py <task>")
        print('Example: python v1_basic.py "create a hello world script"')
        sys.exit(1)

    task = sys.argv[1]
    print(f"Task: {task}")
    result = agent(task)
    print(f"\n[Done]")
