// Monitoring module for HybridPipe
//
// This module provides monitoring functionality for HybridPipe.

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::{Duration, Instant};

#[cfg(test)]
mod tests;

/// MessageStats collects metrics about message processing.
pub struct MessageStats {
    /// Dispatch metrics for each pipe.
    dispatch_metrics: RwLock<HashMap<String, PipeMetrics>>,
    /// Process metrics for each pipe.
    process_metrics: RwLock<HashMap<String, PipeMetrics>>,
    /// Dispatch start times for each pipe.
    dispatch_start_times: RwLock<HashMap<String, Instant>>,
    /// Process start times for each pipe.
    process_start_times: RwLock<HashMap<String, Instant>>,
}

/// PipeMetrics contains metrics for a single pipe.
#[derive(Debug, Clone, Default)]
pub struct PipeMetrics {
    /// Total number of messages.
    pub total_count: u64,
    /// Number of successful messages.
    pub success_count: u64,
    /// Number of failed messages.
    pub error_count: u64,
    /// Total processing time.
    pub total_time: Duration,
    /// Minimum processing time.
    pub min_time: Option<Duration>,
    /// Maximum processing time.
    pub max_time: Option<Duration>,
    /// Total message size in bytes.
    pub total_size: u64,
}

impl MessageStats {
    /// Create a new message stats collector.
    pub fn new() -> Self {
        Self {
            dispatch_metrics: RwLock::new(HashMap::new()),
            process_metrics: RwLock::new(HashMap::new()),
            dispatch_start_times: RwLock::new(HashMap::new()),
            process_start_times: RwLock::new(HashMap::new()),
        }
    }

    /// Record the start of a dispatch operation.
    pub fn record_dispatch_start(&self, pipe: &str) {
        let mut start_times = self.dispatch_start_times.write().unwrap();
        start_times.insert(pipe.to_string(), Instant::now());
    }

    /// Record the end of a dispatch operation.
    pub fn record_dispatch_end(&self, pipe: &str, success: bool) {
        let now = Instant::now();

        // Get the start time
        let start_time = {
            let mut start_times = self.dispatch_start_times.write().unwrap();
            start_times.remove(pipe).unwrap_or(now)
        };

        // Calculate the duration
        let duration = now.duration_since(start_time);

        // Update the metrics
        let mut metrics = self.dispatch_metrics.write().unwrap();
        let pipe_metrics = metrics
            .entry(pipe.to_string())
            .or_insert_with(PipeMetrics::default);

        pipe_metrics.total_count += 1;
        if success {
            pipe_metrics.success_count += 1;
        } else {
            pipe_metrics.error_count += 1;
        }

        pipe_metrics.total_time += duration;

        if let Some(min_time) = pipe_metrics.min_time {
            if duration < min_time {
                pipe_metrics.min_time = Some(duration);
            }
        } else {
            pipe_metrics.min_time = Some(duration);
        }

        if let Some(max_time) = pipe_metrics.max_time {
            if duration > max_time {
                pipe_metrics.max_time = Some(duration);
            }
        } else {
            pipe_metrics.max_time = Some(duration);
        }
    }

    /// Record the start of a process operation.
    pub fn record_process_start(&self, pipe: &str, size: usize) {
        let mut start_times = self.process_start_times.write().unwrap();
        start_times.insert(pipe.to_string(), Instant::now());

        // Update the total size
        let mut metrics = self.process_metrics.write().unwrap();
        let pipe_metrics = metrics
            .entry(pipe.to_string())
            .or_insert_with(PipeMetrics::default);
        pipe_metrics.total_size += size as u64;
    }

    /// Record the end of a process operation.
    pub fn record_process_end(&self, pipe: &str, success: bool) {
        let now = Instant::now();

        // Get the start time
        let start_time = {
            let mut start_times = self.process_start_times.write().unwrap();
            start_times.remove(pipe).unwrap_or(now)
        };

        // Calculate the duration
        let duration = now.duration_since(start_time);

        // Update the metrics
        let mut metrics = self.process_metrics.write().unwrap();
        let pipe_metrics = metrics
            .entry(pipe.to_string())
            .or_insert_with(PipeMetrics::default);

        pipe_metrics.total_count += 1;
        if success {
            pipe_metrics.success_count += 1;
        } else {
            pipe_metrics.error_count += 1;
        }

        pipe_metrics.total_time += duration;

        if let Some(min_time) = pipe_metrics.min_time {
            if duration < min_time {
                pipe_metrics.min_time = Some(duration);
            }
        } else {
            pipe_metrics.min_time = Some(duration);
        }

        if let Some(max_time) = pipe_metrics.max_time {
            if duration > max_time {
                pipe_metrics.max_time = Some(duration);
            }
        } else {
            pipe_metrics.max_time = Some(duration);
        }
    }

    /// Get the dispatch metrics for all pipes.
    pub fn get_dispatch_metrics(&self) -> HashMap<String, PipeMetrics> {
        let metrics = self.dispatch_metrics.read().unwrap();
        metrics.clone()
    }

    /// Get the process metrics for all pipes.
    pub fn get_process_metrics(&self) -> HashMap<String, PipeMetrics> {
        let metrics = self.process_metrics.read().unwrap();
        metrics.clone()
    }

    /// Get the dispatch metrics for a specific pipe.
    pub fn get_pipe_dispatch_metrics(&self, pipe: &str) -> Option<PipeMetrics> {
        let metrics = self.dispatch_metrics.read().unwrap();
        metrics.get(pipe).cloned()
    }

    /// Get the process metrics for a specific pipe.
    pub fn get_pipe_process_metrics(&self, pipe: &str) -> Option<PipeMetrics> {
        let metrics = self.process_metrics.read().unwrap();
        metrics.get(pipe).cloned()
    }

    /// Reset all metrics.
    pub fn reset(&self) {
        let mut dispatch_metrics = self.dispatch_metrics.write().unwrap();
        dispatch_metrics.clear();

        let mut process_metrics = self.process_metrics.write().unwrap();
        process_metrics.clear();

        let mut dispatch_start_times = self.dispatch_start_times.write().unwrap();
        dispatch_start_times.clear();

        let mut process_start_times = self.process_start_times.write().unwrap();
        process_start_times.clear();
    }
}

impl Default for MessageStats {
    fn default() -> Self {
        Self::new()
    }
}

/// Create a new message stats collector.
pub fn new_message_stats() -> Arc<MessageStats> {
    Arc::new(MessageStats::new())
}
