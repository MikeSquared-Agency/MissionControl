use std::ffi::{CStr, CString};
use std::os::raw::c_char;

use workflow::{WorkflowEngine, Task, TaskStatus, Phase, GateStatus};
use knowledge::{KnowledgeManager, Handoff, BudgetStatus};
use runtime::{HealthMonitor, HealthStatus};

// ============================================================================
// String Management
// ============================================================================

/// Free a string that was allocated by Rust
#[no_mangle]
pub extern "C" fn missioncontrol_free_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        unsafe {
            drop(CString::from_raw(ptr));
        }
    }
}

/// Helper to convert Rust string to C string
fn to_c_string(s: &str) -> *mut c_char {
    CString::new(s)
        .map(|cs| cs.into_raw())
        .unwrap_or(std::ptr::null_mut())
}

/// Helper to convert C string to Rust string
fn from_c_string(ptr: *const c_char) -> Option<String> {
    if ptr.is_null() {
        return None;
    }
    unsafe {
        CStr::from_ptr(ptr)
            .to_str()
            .ok()
            .map(|s| s.to_string())
    }
}

// ============================================================================
// Workflow Engine FFI
// ============================================================================

/// Create a new WorkflowEngine
#[no_mangle]
pub extern "C" fn workflow_engine_new() -> *mut WorkflowEngine {
    Box::into_raw(Box::new(WorkflowEngine::new()))
}

/// Free a WorkflowEngine
#[no_mangle]
pub extern "C" fn workflow_engine_free(ptr: *mut WorkflowEngine) {
    if !ptr.is_null() {
        unsafe {
            drop(Box::from_raw(ptr));
        }
    }
}

/// Get current phase as JSON string
#[no_mangle]
pub extern "C" fn workflow_engine_current_phase(ptr: *const WorkflowEngine) -> *mut c_char {
    if ptr.is_null() {
        return std::ptr::null_mut();
    }

    let engine = unsafe { &*ptr };
    let phase = engine.current_phase();
    let json = serde_json::json!({
        "phase": phase.as_str()
    });

    to_c_string(&json.to_string())
}

/// Create a task from JSON, returns task ID or error
#[no_mangle]
pub extern "C" fn workflow_engine_create_task(
    ptr: *mut WorkflowEngine,
    task_json: *const c_char,
) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string(r#"{"error": "null engine pointer"}"#);
    }

    let json_str = match from_c_string(task_json) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid task JSON"}"#),
    };

    let task: Task = match serde_json::from_str(&json_str) {
        Ok(t) => t,
        Err(e) => return to_c_string(&format!(r#"{{"error": "{}"}}"#, e)),
    };

    let engine = unsafe { &mut *ptr };
    let id = engine.create_task(task);

    to_c_string(&format!(r#"{{"task_id": "{}"}}"#, id))
}

/// Get ready tasks as JSON array
#[no_mangle]
pub extern "C" fn workflow_engine_get_ready_tasks(ptr: *const WorkflowEngine) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string("[]");
    }

    let engine = unsafe { &*ptr };
    let tasks = engine.get_ready_tasks();

    match serde_json::to_string(&tasks) {
        Ok(json) => to_c_string(&json),
        Err(_) => to_c_string("[]"),
    }
}

/// Get all tasks as JSON array
#[no_mangle]
pub extern "C" fn workflow_engine_get_all_tasks(ptr: *const WorkflowEngine) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string("[]");
    }

    let engine = unsafe { &*ptr };
    let tasks = engine.all_tasks();

    match serde_json::to_string(&tasks) {
        Ok(json) => to_c_string(&json),
        Err(_) => to_c_string("[]"),
    }
}

/// Update task status
#[no_mangle]
pub extern "C" fn workflow_engine_update_task_status(
    ptr: *mut WorkflowEngine,
    task_id: *const c_char,
    status_json: *const c_char,
) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string(r#"{"error": "null engine pointer"}"#);
    }

    let id = match from_c_string(task_id) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid task ID"}"#),
    };

    let status_str = match from_c_string(status_json) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid status JSON"}"#),
    };

    let status: TaskStatus = match serde_json::from_str(&status_str) {
        Ok(s) => s,
        Err(e) => return to_c_string(&format!(r#"{{"error": "{}"}}"#, e)),
    };

    let engine = unsafe { &mut *ptr };
    match engine.update_task_status(&id, status) {
        Ok(()) => to_c_string(r#"{"success": true}"#),
        Err(e) => to_c_string(&format!(r#"{{"error": "{}"}}"#, e)),
    }
}

/// Check gate status for a phase
#[no_mangle]
pub extern "C" fn workflow_engine_check_gate(
    ptr: *const WorkflowEngine,
    phase_str: *const c_char,
) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string(r#"{"error": "null engine pointer"}"#);
    }

    let phase_name = match from_c_string(phase_str) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid phase"}"#),
    };

    let phase: Phase = match serde_json::from_str(&format!(r#""{}""#, phase_name)) {
        Ok(p) => p,
        Err(_) => return to_c_string(r#"{"error": "unknown phase"}"#),
    };

    let engine = unsafe { &*ptr };
    let status = engine.check_gate(phase);

    let status_str = match status {
        GateStatus::Open => "open",
        GateStatus::Closed => "closed",
        GateStatus::AwaitingApproval => "awaiting_approval",
    };

    to_c_string(&format!(r#"{{"status": "{}"}}"#, status_str))
}

/// Approve a gate
#[no_mangle]
pub extern "C" fn workflow_engine_approve_gate(
    ptr: *mut WorkflowEngine,
    phase_str: *const c_char,
    approved_by: *const c_char,
) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string(r#"{"error": "null engine pointer"}"#);
    }

    let phase_name = match from_c_string(phase_str) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid phase"}"#),
    };

    let by = match from_c_string(approved_by) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid approver"}"#),
    };

    let phase: Phase = match serde_json::from_str(&format!(r#""{}""#, phase_name)) {
        Ok(p) => p,
        Err(_) => return to_c_string(r#"{"error": "unknown phase"}"#),
    };

    let engine = unsafe { &mut *ptr };
    match engine.approve_gate(phase, &by) {
        Ok(()) => to_c_string(r#"{"success": true}"#),
        Err(e) => to_c_string(&format!(r#"{{"error": "{}"}}"#, e)),
    }
}

/// Serialize engine to JSON
#[no_mangle]
pub extern "C" fn workflow_engine_to_json(ptr: *const WorkflowEngine) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string("{}");
    }

    let engine = unsafe { &*ptr };
    to_c_string(&engine.to_json())
}

/// Deserialize engine from JSON
#[no_mangle]
pub extern "C" fn workflow_engine_from_json(json: *const c_char) -> *mut WorkflowEngine {
    let json_str = match from_c_string(json) {
        Some(s) => s,
        None => return std::ptr::null_mut(),
    };

    match WorkflowEngine::from_json(&json_str) {
        Ok(engine) => Box::into_raw(Box::new(engine)),
        Err(_) => std::ptr::null_mut(),
    }
}

// ============================================================================
// Knowledge Manager FFI
// ============================================================================

/// Create a new KnowledgeManager
#[no_mangle]
pub extern "C" fn knowledge_manager_new() -> *mut KnowledgeManager {
    Box::into_raw(Box::new(KnowledgeManager::new()))
}

/// Free a KnowledgeManager
#[no_mangle]
pub extern "C" fn knowledge_manager_free(ptr: *mut KnowledgeManager) {
    if !ptr.is_null() {
        unsafe {
            drop(Box::from_raw(ptr));
        }
    }
}

/// Count tokens in text
#[no_mangle]
pub extern "C" fn knowledge_manager_count_tokens(
    ptr: *const KnowledgeManager,
    text: *const c_char,
) -> usize {
    if ptr.is_null() {
        return 0;
    }

    let text_str = match from_c_string(text) {
        Some(s) => s,
        None => return 0,
    };

    let manager = unsafe { &*ptr };
    manager.count_tokens(&text_str)
}

/// Create a token budget for a worker
#[no_mangle]
pub extern "C" fn knowledge_manager_create_budget(
    ptr: *mut KnowledgeManager,
    worker_id: *const c_char,
    budget: usize,
) {
    if ptr.is_null() {
        return;
    }

    let id = match from_c_string(worker_id) {
        Some(s) => s,
        None => return,
    };

    let manager = unsafe { &mut *ptr };
    manager.create_budget(&id, budget);
}

/// Check budget status for a worker
#[no_mangle]
pub extern "C" fn knowledge_manager_check_budget(
    ptr: *const KnowledgeManager,
    worker_id: *const c_char,
) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string(r#"{"error": "null manager pointer"}"#);
    }

    let id = match from_c_string(worker_id) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid worker ID"}"#),
    };

    let manager = unsafe { &*ptr };
    match manager.check_budget(&id) {
        Some(status) => {
            let (status_str, remaining) = match status {
                BudgetStatus::Healthy => ("healthy", None),
                BudgetStatus::Warning { remaining } => ("warning", Some(remaining)),
                BudgetStatus::Critical { remaining } => ("critical", Some(remaining)),
                BudgetStatus::Exceeded => ("exceeded", None),
            };

            if let Some(r) = remaining {
                to_c_string(&format!(r#"{{"status": "{}", "remaining": {}}}"#, status_str, r))
            } else {
                to_c_string(&format!(r#"{{"status": "{}"}}"#, status_str))
            }
        }
        None => to_c_string(r#"{"error": "worker not found"}"#),
    }
}

/// Validate a handoff
#[no_mangle]
pub extern "C" fn knowledge_manager_validate_handoff(
    ptr: *const KnowledgeManager,
    handoff_json: *const c_char,
) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string(r#"{"error": "null manager pointer"}"#);
    }

    let json_str = match from_c_string(handoff_json) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid handoff JSON"}"#),
    };

    let handoff: Handoff = match serde_json::from_str(&json_str) {
        Ok(h) => h,
        Err(e) => return to_c_string(&format!(r#"{{"error": "parse error: {}"}}"#, e)),
    };

    let manager = unsafe { &*ptr };
    match manager.validate_handoff(&handoff) {
        Ok(()) => std::ptr::null_mut(), // null means valid
        Err(e) => to_c_string(&format!(r#"{{"error": "{}"}}"#, e)),
    }
}

// ============================================================================
// Health Monitor FFI
// ============================================================================

/// Create a new HealthMonitor
#[no_mangle]
pub extern "C" fn health_monitor_new() -> *mut HealthMonitor {
    Box::into_raw(Box::new(HealthMonitor::new()))
}

/// Create a new HealthMonitor with custom thresholds
#[no_mangle]
pub extern "C" fn health_monitor_with_thresholds(stuck_ms: u64, idle_ms: u64) -> *mut HealthMonitor {
    Box::into_raw(Box::new(HealthMonitor::with_thresholds(stuck_ms, idle_ms)))
}

/// Free a HealthMonitor
#[no_mangle]
pub extern "C" fn health_monitor_free(ptr: *mut HealthMonitor) {
    if !ptr.is_null() {
        unsafe {
            drop(Box::from_raw(ptr));
        }
    }
}

/// Register a worker
#[no_mangle]
pub extern "C" fn health_monitor_register_worker(
    ptr: *mut HealthMonitor,
    worker_id: *const c_char,
) {
    if ptr.is_null() {
        return;
    }

    let id = match from_c_string(worker_id) {
        Some(s) => s,
        None => return,
    };

    let monitor = unsafe { &mut *ptr };
    monitor.register_worker(&id);
}

/// Unregister a worker
#[no_mangle]
pub extern "C" fn health_monitor_unregister_worker(
    ptr: *mut HealthMonitor,
    worker_id: *const c_char,
) {
    if ptr.is_null() {
        return;
    }

    let id = match from_c_string(worker_id) {
        Some(s) => s,
        None => return,
    };

    let monitor = unsafe { &mut *ptr };
    monitor.unregister_worker(&id);
}

/// Mark activity for a worker
#[no_mangle]
pub extern "C" fn health_monitor_mark_activity(
    ptr: *mut HealthMonitor,
    worker_id: *const c_char,
) {
    if ptr.is_null() {
        return;
    }

    let id = match from_c_string(worker_id) {
        Some(s) => s,
        None => return,
    };

    let monitor = unsafe { &mut *ptr };
    monitor.mark_activity(&id);
}

/// Mark a tool call for a worker
#[no_mangle]
pub extern "C" fn health_monitor_mark_tool_call(
    ptr: *mut HealthMonitor,
    worker_id: *const c_char,
) {
    if ptr.is_null() {
        return;
    }

    let id = match from_c_string(worker_id) {
        Some(s) => s,
        None => return,
    };

    let monitor = unsafe { &mut *ptr };
    monitor.mark_tool_call(&id);
}

/// Check health status for a worker
#[no_mangle]
pub extern "C" fn health_monitor_check_health(
    ptr: *const HealthMonitor,
    worker_id: *const c_char,
) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string(r#"{"error": "null monitor pointer"}"#);
    }

    let id = match from_c_string(worker_id) {
        Some(s) => s,
        None => return to_c_string(r#"{"error": "invalid worker ID"}"#),
    };

    let monitor = unsafe { &*ptr };
    match monitor.check_health(&id) {
        Some(status) => {
            let json = match status {
                HealthStatus::Healthy => r#"{"status": "healthy"}"#.to_string(),
                HealthStatus::Idle { since_ms } => format!(r#"{{"status": "idle", "since_ms": {}}}"#, since_ms),
                HealthStatus::Stuck { since_ms } => format!(r#"{{"status": "stuck", "since_ms": {}}}"#, since_ms),
                HealthStatus::Unresponsive => r#"{"status": "unresponsive"}"#.to_string(),
                HealthStatus::Dead => r#"{"status": "dead"}"#.to_string(),
            };
            to_c_string(&json)
        }
        None => to_c_string(r#"{"error": "worker not found"}"#),
    }
}

/// Get all stuck workers as JSON array
#[no_mangle]
pub extern "C" fn health_monitor_get_stuck_workers(ptr: *const HealthMonitor) -> *mut c_char {
    if ptr.is_null() {
        return to_c_string("[]");
    }

    let monitor = unsafe { &*ptr };
    let stuck = monitor.get_stuck_workers();

    match serde_json::to_string(&stuck) {
        Ok(json) => to_c_string(&json),
        Err(_) => to_c_string("[]"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_workflow_engine_lifecycle() {
        let engine = workflow_engine_new();
        assert!(!engine.is_null());

        let phase = workflow_engine_current_phase(engine);
        assert!(!phase.is_null());

        // Clean up
        missioncontrol_free_string(phase);
        workflow_engine_free(engine);
    }

    #[test]
    fn test_knowledge_manager_lifecycle() {
        let manager = knowledge_manager_new();
        assert!(!manager.is_null());

        let text = CString::new("hello world").unwrap();
        let count = knowledge_manager_count_tokens(manager, text.as_ptr());
        assert!(count > 0);

        knowledge_manager_free(manager);
    }

    #[test]
    fn test_health_monitor_lifecycle() {
        let monitor = health_monitor_new();
        assert!(!monitor.is_null());

        let worker_id = CString::new("worker-1").unwrap();
        health_monitor_register_worker(monitor, worker_id.as_ptr());

        let health = health_monitor_check_health(monitor, worker_id.as_ptr());
        assert!(!health.is_null());

        missioncontrol_free_string(health);
        health_monitor_free(monitor);
    }
}
