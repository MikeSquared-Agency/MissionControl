use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Stage {
    Discovery,
    Goal,
    Requirements,
    Planning,
    Design,
    Implement,
    Verify,
    Validate,
    Document,
    Release,
}

impl Stage {
    pub fn next(&self) -> Option<Stage> {
        match self {
            Stage::Discovery => Some(Stage::Goal),
            Stage::Goal => Some(Stage::Requirements),
            Stage::Requirements => Some(Stage::Planning),
            Stage::Planning => Some(Stage::Design),
            Stage::Design => Some(Stage::Implement),
            Stage::Implement => Some(Stage::Verify),
            Stage::Verify => Some(Stage::Validate),
            Stage::Validate => Some(Stage::Document),
            Stage::Document => Some(Stage::Release),
            Stage::Release => None,
        }
    }

    pub fn all() -> &'static [Stage] {
        &[
            Stage::Discovery,
            Stage::Goal,
            Stage::Requirements,
            Stage::Planning,
            Stage::Design,
            Stage::Implement,
            Stage::Verify,
            Stage::Validate,
            Stage::Document,
            Stage::Release,
        ]
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Stage::Discovery => "discovery",
            Stage::Goal => "goal",
            Stage::Requirements => "requirements",
            Stage::Planning => "planning",
            Stage::Design => "design",
            Stage::Implement => "implement",
            Stage::Verify => "verify",
            Stage::Validate => "validate",
            Stage::Document => "document",
            Stage::Release => "release",
        }
    }
}

impl Default for Stage {
    fn default() -> Self {
        Stage::Discovery
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_stage_next() {
        assert_eq!(Stage::Discovery.next(), Some(Stage::Goal));
        assert_eq!(Stage::Goal.next(), Some(Stage::Requirements));
        assert_eq!(Stage::Requirements.next(), Some(Stage::Planning));
        assert_eq!(Stage::Planning.next(), Some(Stage::Design));
        assert_eq!(Stage::Design.next(), Some(Stage::Implement));
        assert_eq!(Stage::Implement.next(), Some(Stage::Verify));
        assert_eq!(Stage::Verify.next(), Some(Stage::Validate));
        assert_eq!(Stage::Validate.next(), Some(Stage::Document));
        assert_eq!(Stage::Document.next(), Some(Stage::Release));
        assert_eq!(Stage::Release.next(), None);
    }

    #[test]
    fn test_stage_all() {
        let all = Stage::all();
        assert_eq!(all.len(), 10);
        assert_eq!(all[0], Stage::Discovery);
        assert_eq!(all[9], Stage::Release);
    }

    #[test]
    fn test_stage_serialization() {
        let stage = Stage::Implement;
        let json = serde_json::to_string(&stage).unwrap();
        assert_eq!(json, "\"implement\"");

        let parsed: Stage = serde_json::from_str("\"design\"").unwrap();
        assert_eq!(parsed, Stage::Design);

        let parsed: Stage = serde_json::from_str("\"discovery\"").unwrap();
        assert_eq!(parsed, Stage::Discovery);

        let parsed: Stage = serde_json::from_str("\"validate\"").unwrap();
        assert_eq!(parsed, Stage::Validate);
    }

    #[test]
    fn test_stage_as_str() {
        assert_eq!(Stage::Discovery.as_str(), "discovery");
        assert_eq!(Stage::Goal.as_str(), "goal");
        assert_eq!(Stage::Requirements.as_str(), "requirements");
        assert_eq!(Stage::Planning.as_str(), "planning");
        assert_eq!(Stage::Design.as_str(), "design");
        assert_eq!(Stage::Implement.as_str(), "implement");
        assert_eq!(Stage::Verify.as_str(), "verify");
        assert_eq!(Stage::Validate.as_str(), "validate");
        assert_eq!(Stage::Document.as_str(), "document");
        assert_eq!(Stage::Release.as_str(), "release");
    }

    #[test]
    fn test_stage_default() {
        assert_eq!(Stage::default(), Stage::Discovery);
    }
}
