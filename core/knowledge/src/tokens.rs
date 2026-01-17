use tiktoken_rs::cl100k_base;

pub struct TokenCounter {
    bpe: tiktoken_rs::CoreBPE,
}

impl TokenCounter {
    pub fn new() -> Self {
        Self {
            bpe: cl100k_base().expect("Failed to initialize tiktoken"),
        }
    }

    pub fn count(&self, text: &str) -> usize {
        self.bpe.encode_with_special_tokens(text).len()
    }
}

impl Default for TokenCounter {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_token_counting() {
        let counter = TokenCounter::new();

        // Simple test - "hello world" should be 2 tokens
        let count = counter.count("hello world");
        assert!(count > 0);

        // Empty string should be 0 tokens
        assert_eq!(counter.count(""), 0);
    }

    #[test]
    fn test_longer_text() {
        let counter = TokenCounter::new();
        let text = "This is a longer piece of text that should have more tokens than a simple hello world.";
        let count = counter.count(text);
        assert!(count > 10);
    }
}
