#!/usr/bin/env python3
"""Minimal agent (~50 lines). Proves agents are tiny."""

import subprocess
import sys
import anthropic

client = anthropic.Anthropic()

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
    }
]


def execute(tool_name: str, tool_input: dict) -> str:
    if tool_name == "bash":
        result = subprocess.run(
            tool_input["command"], shell=True, capture_output=True, text=True
        )
        return result.stdout + result.stderr
    return f"Unknown tool: {tool_name}"


def agent(task: str) -> str:
    messages = [{"role": "user", "content": task}]

    while True:
        response = client.messages.create(
            model="claude-sonnet-4-20250514",
            max_tokens=4096,
            tools=tools,
            messages=messages,
        )

        # If no tool use, we're done
        if response.stop_reason != "tool_use":
            return "".join(b.text for b in response.content if hasattr(b, "text"))

        # Execute tool calls and collect results
        tool_results = []
        for block in response.content:
            if block.type == "tool_use":
                print(f"$ {block.input.get('command', block.name)}")
                result = execute(block.name, block.input)
                print(result)
                tool_results.append(
                    {"type": "tool_result", "tool_use_id": block.id, "content": result}
                )

        messages.append({"role": "assistant", "content": response.content})
        messages.append({"role": "user", "content": tool_results})


if __name__ == "__main__":
    task = sys.argv[1] if len(sys.argv) > 1 else "List the files in the current directory"
    print(agent(task))
