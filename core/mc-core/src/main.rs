use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use knowledge::{Handoff, HandoffStatus, TokenCounter};
use serde::{Deserialize, Serialize};
use std::fs;
use std::io::{self, Read};
use std::path::PathBuf;
use workflow::{Gate, GateStatus, Phase};

#[derive(Parser)]
#[command(name = "mc-core")]
#[command(about = "MissionControl core CLI - validation, gate checking, token counting")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Validate a handoff JSON file
    ValidateHandoff {
        /// Path to the handoff JSON file
        file: PathBuf,
    },
    /// Check gate criteria for a phase
    CheckGate {
        /// Phase to check (idea, design, implement, verify, document, release)
        phase: String,
        /// Path to the .mission directory
        #[arg(long, default_value = ".mission")]
        mission_dir: PathBuf,
    },
    /// Count tokens in text (from file or stdin)
    CountTokens {
        /// Path to file, or "-" for stdin (default: stdin)
        #[arg(default_value = "-")]
        source: String,
    },
}

#[derive(Debug, Serialize)]
struct ValidationResult {
    valid: bool,
    errors: Vec<String>,
    warnings: Vec<String>,
}

#[derive(Debug, Serialize)]
struct GateCheckResult {
    phase: String,
    status: String,
    criteria: Vec<CriterionResult>,
    can_approve: bool,
}

#[derive(Debug, Serialize)]
struct CriterionResult {
    description: String,
    satisfied: bool,
}

#[derive(Debug, Serialize)]
struct TokenCountResult {
    tokens: usize,
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    match cli.command {
        Commands::ValidateHandoff { file } => {
            let result = validate_handoff(&file)?;
            println!("{}", serde_json::to_string_pretty(&result)?);
            if !result.valid {
                std::process::exit(1);
            }
        }
        Commands::CheckGate { phase, mission_dir } => {
            let result = check_gate(&phase, &mission_dir)?;
            println!("{}", serde_json::to_string_pretty(&result)?);
        }
        Commands::CountTokens { source } => {
            let result = count_tokens(&source)?;
            println!("{}", serde_json::to_string(&result)?);
        }
    }

    Ok(())
}

fn validate_handoff(file: &PathBuf) -> Result<ValidationResult> {
    let mut errors = Vec::new();
    let mut warnings = Vec::new();

    // Read file
    let content = fs::read_to_string(file)
        .with_context(|| format!("Failed to read file: {}", file.display()))?;

    // Parse JSON
    let handoff: Handoff = match serde_json::from_str(&content) {
        Ok(h) => h,
        Err(e) => {
            errors.push(format!("Invalid JSON: {}", e));
            return Ok(ValidationResult {
                valid: false,
                errors,
                warnings,
            });
        }
    };

    // Validate required fields
    if handoff.task_id.is_empty() {
        errors.push("task_id is required".to_string());
    }

    if handoff.worker_id.is_empty() {
        errors.push("worker_id is required".to_string());
    }

    // Semantic validations
    if handoff.findings.is_empty() {
        warnings.push("No findings reported - consider documenting discoveries".to_string());
    }

    // Check for blockers
    if let HandoffStatus::Blocked(reason) = &handoff.status {
        if reason.is_empty() {
            errors.push("Blocked status requires a reason".to_string());
        }
    }

    // Check artifacts exist (warning only)
    for artifact in &handoff.artifacts {
        if !PathBuf::from(artifact).exists() {
            warnings.push(format!("Artifact not found: {}", artifact));
        }
    }

    // Validate findings
    for (i, finding) in handoff.findings.iter().enumerate() {
        if finding.summary.is_empty() {
            errors.push(format!("Finding {} has empty summary", i));
        }
        if finding.summary.len() > 500 {
            warnings.push(format!("Finding {} summary is very long (>500 chars)", i));
        }
    }

    Ok(ValidationResult {
        valid: errors.is_empty(),
        errors,
        warnings,
    })
}

fn check_gate(phase_str: &str, mission_dir: &PathBuf) -> Result<GateCheckResult> {
    // Parse phase
    let phase: Phase = serde_json::from_str(&format!("\"{}\"", phase_str))
        .with_context(|| format!("Invalid phase: {}. Valid: idea, design, implement, verify, document, release", phase_str))?;

    // Try to read existing gate state
    let gates_file = mission_dir.join("state/gates.json");
    let gate = if gates_file.exists() {
        let content = fs::read_to_string(&gates_file)
            .with_context(|| format!("Failed to read gates file: {}", gates_file.display()))?;

        #[derive(Deserialize)]
        struct GatesFile {
            gates: std::collections::HashMap<String, GateState>,
        }

        #[derive(Deserialize)]
        struct GateState {
            status: String,
            criteria: Vec<String>,
            approved_at: Option<String>,
        }

        let gates: GatesFile = serde_json::from_str(&content)?;

        if let Some(state) = gates.gates.get(phase_str) {
            // Build gate from state
            let mut gate = Gate::new(phase);
            // Map criteria - mark as satisfied if status indicates completion
            for (i, criterion) in gate.criteria.iter_mut().enumerate() {
                // Check if we have enough criteria in state
                if i < state.criteria.len() {
                    // For now, consider criteria satisfied if gate is awaiting_approval or approved
                    if state.status == "awaiting_approval" || state.status == "approved" {
                        criterion.satisfy();
                    }
                }
            }
            if state.approved_at.is_some() {
                gate.approve("system");
            }
            gate
        } else {
            Gate::new(phase)
        }
    } else {
        Gate::new(phase)
    };

    let criteria: Vec<CriterionResult> = gate
        .criteria
        .iter()
        .map(|c| CriterionResult {
            description: c.description.clone(),
            satisfied: c.satisfied,
        })
        .collect();

    let status = match gate.status {
        GateStatus::Open => "open",
        GateStatus::Closed => "closed",
        GateStatus::AwaitingApproval => "awaiting_approval",
    };

    Ok(GateCheckResult {
        phase: phase_str.to_string(),
        status: status.to_string(),
        criteria,
        can_approve: gate.all_criteria_satisfied() && gate.approved_at.is_none(),
    })
}

fn count_tokens(source: &str) -> Result<TokenCountResult> {
    let content = if source == "-" {
        // Read from stdin
        let mut buffer = String::new();
        io::stdin()
            .read_to_string(&mut buffer)
            .context("Failed to read from stdin")?;
        buffer
    } else {
        // Read from file
        fs::read_to_string(source)
            .with_context(|| format!("Failed to read file: {}", source))?
    };

    let counter = TokenCounter::new();
    let tokens = counter.count(&content);

    Ok(TokenCountResult { tokens })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_validate_handoff_valid() {
        let handoff = r#"{
            "task_id": "task-1",
            "worker_id": "worker-1",
            "status": "complete",
            "findings": [
                {
                    "finding_type": "discovery",
                    "summary": "Found existing implementation"
                }
            ],
            "artifacts": [],
            "open_questions": [],
            "context_for_successor": null,
            "timestamp": 1234567890
        }"#;

        let mut file = NamedTempFile::new().unwrap();
        file.write_all(handoff.as_bytes()).unwrap();

        let result = validate_handoff(&file.path().to_path_buf()).unwrap();
        assert!(result.valid);
        assert!(result.errors.is_empty());
    }

    #[test]
    fn test_validate_handoff_invalid() {
        let handoff = r#"{
            "task_id": "",
            "worker_id": "worker-1",
            "status": "complete",
            "findings": [],
            "artifacts": [],
            "open_questions": [],
            "timestamp": 1234567890
        }"#;

        let mut file = NamedTempFile::new().unwrap();
        file.write_all(handoff.as_bytes()).unwrap();

        let result = validate_handoff(&file.path().to_path_buf()).unwrap();
        assert!(!result.valid);
        assert!(result.errors.iter().any(|e| e.contains("task_id")));
    }

    #[test]
    fn test_count_tokens() {
        let content = "Hello world, this is a test.";
        let mut file = NamedTempFile::new().unwrap();
        file.write_all(content.as_bytes()).unwrap();

        let result = count_tokens(file.path().to_str().unwrap()).unwrap();
        assert!(result.tokens > 0);
    }
}
