use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Phase {
    Idea,
    Design,
    Implement,
    Verify,
    Document,
    Release,
}

impl Phase {
    pub fn next(&self) -> Option<Phase> {
        match self {
            Phase::Idea => Some(Phase::Design),
            Phase::Design => Some(Phase::Implement),
            Phase::Implement => Some(Phase::Verify),
            Phase::Verify => Some(Phase::Document),
            Phase::Document => Some(Phase::Release),
            Phase::Release => None,
        }
    }

    pub fn all() -> &'static [Phase] {
        &[
            Phase::Idea,
            Phase::Design,
            Phase::Implement,
            Phase::Verify,
            Phase::Document,
            Phase::Release,
        ]
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Phase::Idea => "idea",
            Phase::Design => "design",
            Phase::Implement => "implement",
            Phase::Verify => "verify",
            Phase::Document => "document",
            Phase::Release => "release",
        }
    }
}

impl Default for Phase {
    fn default() -> Self {
        Phase::Idea
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_phase_next() {
        assert_eq!(Phase::Idea.next(), Some(Phase::Design));
        assert_eq!(Phase::Design.next(), Some(Phase::Implement));
        assert_eq!(Phase::Implement.next(), Some(Phase::Verify));
        assert_eq!(Phase::Verify.next(), Some(Phase::Document));
        assert_eq!(Phase::Document.next(), Some(Phase::Release));
        assert_eq!(Phase::Release.next(), None);
    }

    #[test]
    fn test_phase_all() {
        let all = Phase::all();
        assert_eq!(all.len(), 6);
        assert_eq!(all[0], Phase::Idea);
        assert_eq!(all[5], Phase::Release);
    }

    #[test]
    fn test_phase_serialization() {
        let phase = Phase::Implement;
        let json = serde_json::to_string(&phase).unwrap();
        assert_eq!(json, "\"implement\"");

        let parsed: Phase = serde_json::from_str("\"design\"").unwrap();
        assert_eq!(parsed, Phase::Design);
    }
}
