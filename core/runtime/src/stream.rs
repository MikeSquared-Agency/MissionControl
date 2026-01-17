use serde::Serialize;
use serde_json::Value;

/// Unified event format for the orchestrator and UI
#[derive(Debug, Clone, Serialize)]
pub struct UnifiedEvent {
    #[serde(rename = "type")]
    pub event_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub agent_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub args: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub turn: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tokens: Option<u32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
}

impl UnifiedEvent {
    pub fn new(event_type: impl Into<String>) -> Self {
        UnifiedEvent {
            event_type: event_type.into(),
            agent_id: None,
            content: None,
            tool: None,
            args: None,
            result: None,
            turn: None,
            tokens: None,
            status: None,
            error: None,
        }
    }

    pub fn with_agent_id(mut self, id: impl Into<String>) -> Self {
        self.agent_id = Some(id.into());
        self
    }

    pub fn with_content(mut self, content: impl Into<String>) -> Self {
        self.content = Some(content.into());
        self
    }

    pub fn with_tool(mut self, tool: impl Into<String>, args: Value) -> Self {
        self.tool = Some(tool.into());
        self.args = Some(args);
        self
    }

    pub fn with_result(mut self, result: impl Into<String>) -> Self {
        self.result = Some(result.into());
        self
    }

    pub fn with_turn(mut self, turn: u32) -> Self {
        self.turn = Some(turn);
        self
    }

    pub fn with_tokens(mut self, tokens: u32) -> Self {
        self.tokens = Some(tokens);
        self
    }

    pub fn with_error(mut self, error: impl Into<String>) -> Self {
        self.error = Some(error.into());
        self
    }
}

/// Agent output format type
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AgentFormat {
    Python,
    ClaudeCode,
    Unknown,
}

/// Stream parser for agent output
pub struct StreamParser {
    format: AgentFormat,
    agent_id: String,
    current_turn: u32,
}

impl StreamParser {
    pub fn new(agent_id: impl Into<String>) -> Self {
        StreamParser {
            format: AgentFormat::Unknown,
            agent_id: agent_id.into(),
            current_turn: 0,
        }
    }

    pub fn with_format(mut self, format: AgentFormat) -> Self {
        self.format = format;
        self
    }

    pub fn current_turn(&self) -> u32 {
        self.current_turn
    }

    /// Parse a line and return unified events
    pub fn parse_line(&mut self, line: &str) -> Vec<UnifiedEvent> {
        let trimmed = line.trim();
        if trimmed.is_empty() {
            return vec![];
        }

        // Try to parse as JSON
        if let Ok(json) = serde_json::from_str::<Value>(trimmed) {
            return self.parse_json(json);
        }

        // Not JSON - treat as plain text output
        self.parse_text(trimmed)
    }

    /// Parse JSON input
    fn parse_json(&mut self, json: Value) -> Vec<UnifiedEvent> {
        if self.format == AgentFormat::Unknown {
            self.detect_format(&json);
        }

        match self.format {
            AgentFormat::Python => self.parse_python_json(json),
            AgentFormat::ClaudeCode => self.parse_claude_json(json),
            AgentFormat::Unknown => {
                let events = self.parse_python_json(json.clone());
                if !events.is_empty() {
                    return events;
                }
                self.parse_claude_json(json)
            }
        }
    }

    fn detect_format(&mut self, json: &Value) {
        if let Some(obj) = json.as_object() {
            if let Some(type_val) = obj.get("type").and_then(|v| v.as_str()) {
                match type_val {
                    "assistant" | "user" | "result" | "system" => {
                        self.format = AgentFormat::ClaudeCode;
                        return;
                    }
                    "turn" | "thinking" | "tool_call" | "tool_result" => {
                        self.format = AgentFormat::Python;
                        return;
                    }
                    _ => {}
                }
            }

            if obj.contains_key("message") {
                self.format = AgentFormat::ClaudeCode;
            }
        }
    }

    fn parse_python_json(&mut self, json: Value) -> Vec<UnifiedEvent> {
        let mut events = vec![];

        if let Some(obj) = json.as_object() {
            let event_type = obj.get("type").and_then(|v| v.as_str()).unwrap_or("");

            match event_type {
                "turn" => {
                    if let Some(num) = obj.get("number").and_then(|v| v.as_u64()) {
                        self.current_turn = num as u32;
                        events.push(
                            UnifiedEvent::new("turn")
                                .with_agent_id(&self.agent_id)
                                .with_turn(self.current_turn),
                        );
                    }
                }
                "thinking" => {
                    if let Some(content) = obj.get("content").and_then(|v| v.as_str()) {
                        let mut event = UnifiedEvent::new("thinking")
                            .with_agent_id(&self.agent_id)
                            .with_content(content);
                        if let Some(tokens) = obj.get("tokens").and_then(|v| v.as_u64()) {
                            event = event.with_tokens(tokens as u32);
                        }
                        events.push(event);
                    }
                }
                "tool_call" => {
                    if let Some(tool) = obj.get("tool").and_then(|v| v.as_str()) {
                        let args = obj.get("args").cloned().unwrap_or(Value::Null);
                        events.push(
                            UnifiedEvent::new("tool_call")
                                .with_agent_id(&self.agent_id)
                                .with_tool(tool, args),
                        );
                    }
                }
                "tool_result" => {
                    if let Some(content) = obj.get("content").and_then(|v| v.as_str()) {
                        let mut event = UnifiedEvent::new("tool_result")
                            .with_agent_id(&self.agent_id)
                            .with_result(content);
                        if let Some(tokens) = obj.get("tokens").and_then(|v| v.as_u64()) {
                            event = event.with_tokens(tokens as u32);
                        }
                        events.push(event);
                    }
                }
                _ => {
                    events.push(
                        UnifiedEvent::new("raw")
                            .with_agent_id(&self.agent_id)
                            .with_content(&json.to_string()),
                    );
                }
            }
        }

        events
    }

    fn parse_claude_json(&mut self, json: Value) -> Vec<UnifiedEvent> {
        let mut events = vec![];

        if let Some(obj) = json.as_object() {
            let event_type = obj.get("type").and_then(|v| v.as_str()).unwrap_or("");

            match event_type {
                "assistant" => {
                    if let Some(message) = obj.get("message") {
                        if let Some(content_arr) = message.get("content").and_then(|v| v.as_array()) {
                            for block in content_arr {
                                events.extend(self.parse_claude_content_block(block));
                            }
                        }
                    }
                }
                "content_block_start" => {
                    if let Some(block) = obj.get("content_block") {
                        events.extend(self.parse_claude_content_block(block));
                    }
                }
                "content_block_delta" => {
                    if let Some(delta) = obj.get("delta") {
                        if let Some(text) = delta.get("text").and_then(|v| v.as_str()) {
                            events.push(
                                UnifiedEvent::new("thinking")
                                    .with_agent_id(&self.agent_id)
                                    .with_content(text),
                            );
                        }
                    }
                }
                "result" => {
                    if let Some(result) = obj.get("result").and_then(|v| v.as_str()) {
                        events.push(
                            UnifiedEvent::new("tool_result")
                                .with_agent_id(&self.agent_id)
                                .with_result(result),
                        );
                    } else if let Some(result) = obj.get("result") {
                        events.push(
                            UnifiedEvent::new("tool_result")
                                .with_agent_id(&self.agent_id)
                                .with_result(&result.to_string()),
                        );
                    }
                }
                "message_start" => {
                    self.current_turn += 1;
                    events.push(
                        UnifiedEvent::new("turn")
                            .with_agent_id(&self.agent_id)
                            .with_turn(self.current_turn),
                    );
                }
                "message_stop" => {
                    events.push(
                        UnifiedEvent::new("turn_end")
                            .with_agent_id(&self.agent_id)
                            .with_turn(self.current_turn),
                    );
                }
                "error" => {
                    let error_msg = obj
                        .get("error")
                        .and_then(|e| e.get("message"))
                        .and_then(|v| v.as_str())
                        .unwrap_or("Unknown error");
                    events.push(
                        UnifiedEvent::new("error")
                            .with_agent_id(&self.agent_id)
                            .with_error(error_msg),
                    );
                }
                _ => {
                    events.push(
                        UnifiedEvent::new("raw")
                            .with_agent_id(&self.agent_id)
                            .with_content(&json.to_string()),
                    );
                }
            }
        }

        events
    }

    fn parse_claude_content_block(&self, block: &Value) -> Vec<UnifiedEvent> {
        let mut events = vec![];

        if let Some(obj) = block.as_object() {
            let block_type = obj.get("type").and_then(|v| v.as_str()).unwrap_or("");

            match block_type {
                "text" => {
                    if let Some(text) = obj.get("text").and_then(|v| v.as_str()) {
                        events.push(
                            UnifiedEvent::new("thinking")
                                .with_agent_id(&self.agent_id)
                                .with_content(text),
                        );
                    }
                }
                "tool_use" => {
                    if let Some(name) = obj.get("name").and_then(|v| v.as_str()) {
                        let input = obj.get("input").cloned().unwrap_or(Value::Null);
                        events.push(
                            UnifiedEvent::new("tool_call")
                                .with_agent_id(&self.agent_id)
                                .with_tool(name, input),
                        );
                    }
                }
                "tool_result" => {
                    if let Some(content) = obj.get("content").and_then(|v| v.as_str()) {
                        events.push(
                            UnifiedEvent::new("tool_result")
                                .with_agent_id(&self.agent_id)
                                .with_result(content),
                        );
                    }
                }
                _ => {}
            }
        }

        events
    }

    fn parse_text(&mut self, text: &str) -> Vec<UnifiedEvent> {
        let mut events = vec![];

        // Detect turn markers like "[Turn 1]"
        if text.starts_with("[Turn ") {
            if let Some(end) = text.find(']') {
                if let Ok(num) = text[6..end].parse::<u32>() {
                    self.current_turn = num;
                    events.push(
                        UnifiedEvent::new("turn")
                            .with_agent_id(&self.agent_id)
                            .with_turn(num),
                    );
                    return events;
                }
            }
        }

        // Detect bash commands like "$ ls -la"
        if text.starts_with("$ ") {
            let command = &text[2..];
            events.push(
                UnifiedEvent::new("tool_call")
                    .with_agent_id(&self.agent_id)
                    .with_tool("bash", serde_json::json!({"command": command})),
            );
            return events;
        }

        // Detect tool markers like "[read] path/to/file"
        if text.starts_with('[') {
            if let Some(end) = text.find(']') {
                let tool = &text[1..end];
                let rest = text[end + 1..].trim();
                events.push(
                    UnifiedEvent::new("tool_call")
                        .with_agent_id(&self.agent_id)
                        .with_tool(tool, serde_json::json!({"info": rest})),
                );
                return events;
            }
        }

        // Regular text output
        events.push(
            UnifiedEvent::new("output")
                .with_agent_id(&self.agent_id)
                .with_content(text),
        );

        events
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_python_turn() {
        let mut parser = StreamParser::new("test");
        let events = parser.parse_line(r#"{"type":"turn","number":1}"#);
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].event_type, "turn");
        assert_eq!(events[0].turn, Some(1));
    }

    #[test]
    fn test_parse_python_tool_call() {
        let mut parser = StreamParser::new("test");
        let events = parser.parse_line(r#"{"type":"tool_call","tool":"bash","args":{"command":"ls"}}"#);
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].event_type, "tool_call");
        assert_eq!(events[0].tool, Some("bash".to_string()));
    }

    #[test]
    fn test_parse_text_turn() {
        let mut parser = StreamParser::new("test");
        let events = parser.parse_line("[Turn 1]");
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].event_type, "turn");
        assert_eq!(events[0].turn, Some(1));
    }

    #[test]
    fn test_parse_text_bash() {
        let mut parser = StreamParser::new("test");
        let events = parser.parse_line("$ ls -la");
        assert_eq!(events.len(), 1);
        assert_eq!(events[0].event_type, "tool_call");
        assert_eq!(events[0].tool, Some("bash".to_string()));
    }

    #[test]
    fn test_parse_empty_line() {
        let mut parser = StreamParser::new("test");
        let events = parser.parse_line("");
        assert!(events.is_empty());
    }

    #[test]
    fn test_with_format() {
        let parser = StreamParser::new("test").with_format(AgentFormat::Python);
        assert_eq!(parser.format, AgentFormat::Python);
    }
}
