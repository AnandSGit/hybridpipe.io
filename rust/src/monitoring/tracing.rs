// Tracing module for HybridPipe
//
// This module provides message tracing functionality for HybridPipe.

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use log::debug;
use serde::{Deserialize, Serialize};
use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, RwLock};
// Time imports removed as they're not used
use uuid::Uuid;

use crate::core::{context, BrokerType, Error, HybridPipe, Process, Result};
// Serialization imports removed as they're not used

/// Span represents a trace span.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Span {
    /// Unique identifier for the span
    pub id: String,
    /// Parent span ID
    pub parent_id: Option<String>,
    /// Trace ID
    pub trace_id: String,
    /// Name of the span
    pub name: String,
    /// Start time
    pub start_time: DateTime<Utc>,
    /// End time
    pub end_time: Option<DateTime<Utc>>,
    /// Duration in milliseconds
    pub duration_ms: Option<u64>,
    /// Tags
    pub tags: HashMap<String, String>,
    /// Logs
    pub logs: Vec<Log>,
}

/// Log represents a log entry in a span.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Log {
    /// Timestamp
    pub timestamp: DateTime<Utc>,
    /// Fields
    pub fields: HashMap<String, String>,
}

/// Trace represents a complete trace.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Trace {
    /// Unique identifier for the trace
    pub id: String,
    /// Name of the trace
    pub name: String,
    /// Protocol used
    pub protocol: String,
    /// Pipe name
    pub pipe: String,
    /// Start time
    pub start_time: DateTime<Utc>,
    /// End time
    pub end_time: Option<DateTime<Utc>>,
    /// Duration in milliseconds
    pub duration_ms: Option<u64>,
    /// Status (success or error)
    pub status: TraceStatus,
    /// Error message (if status is error)
    pub error: Option<String>,
    /// Spans
    pub spans: Vec<Span>,
    /// Tags
    pub tags: HashMap<String, String>,
}

/// TraceStatus represents the status of a trace.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum TraceStatus {
    /// Success
    Success,
    /// Error
    Error,
}

/// Tracer is responsible for creating and managing traces.
pub trait Tracer: Send + Sync {
    /// Start a new trace.
    fn start_trace(&self, protocol: &str, pipe: &str) -> Arc<Trace>;

    /// End a trace.
    fn end_trace(&self, trace: Arc<Trace>, status: TraceStatus, error: Option<String>);

    /// Get all traces.
    fn get_traces(&self) -> Vec<Arc<Trace>>;

    /// Clear all traces.
    fn clear_traces(&self);
}

/// InMemoryTracer is an in-memory implementation of the Tracer trait.
pub struct InMemoryTracer {
    /// Traces
    traces: RwLock<Vec<Arc<Trace>>>,
    /// Maximum number of traces to keep
    max_traces: usize,
}

impl InMemoryTracer {
    /// Create a new in-memory tracer.
    pub fn new(max_traces: usize) -> Self {
        Self {
            traces: RwLock::new(Vec::new()),
            max_traces,
        }
    }
}

impl Tracer for InMemoryTracer {
    fn start_trace(&self, protocol: &str, pipe: &str) -> Arc<Trace> {
        let trace = Trace {
            id: Uuid::new_v4().to_string(),
            name: format!("{}/{}", protocol, pipe),
            protocol: protocol.to_string(),
            pipe: pipe.to_string(),
            start_time: Utc::now(),
            end_time: None,
            duration_ms: None,
            status: TraceStatus::Success,
            error: None,
            spans: Vec::new(),
            tags: HashMap::new(),
        };

        let trace = Arc::new(trace);
        debug!("Started trace: {}", trace.id);
        trace
    }

    fn end_trace(&self, trace: Arc<Trace>, status: TraceStatus, error: Option<String>) {
        // Create a mutable copy of the trace
        let mut trace_mut = (*trace).clone();
        trace_mut.end_time = Some(Utc::now());
        trace_mut.status = status;
        trace_mut.error = error;

        // Calculate duration
        if let Some(end_time) = trace_mut.end_time {
            let duration = end_time
                .signed_duration_since(trace_mut.start_time)
                .num_milliseconds();
            trace_mut.duration_ms = Some(duration as u64);
        }

        // Store the trace
        let trace = Arc::new(trace_mut);
        let mut traces = self.traces.write().unwrap();
        traces.push(trace.clone());

        // Trim the traces if necessary
        if self.max_traces > 0 && traces.len() > self.max_traces {
            // Create a new vector with only the most recent traces
            let start_index = traces.len() - self.max_traces;
            let new_traces = traces.iter().skip(start_index).cloned().collect::<Vec<_>>();
            *traces = new_traces;
        }

        debug!("Ended trace: {}", trace.id);
    }

    fn get_traces(&self) -> Vec<Arc<Trace>> {
        let traces = self.traces.read().unwrap();
        traces.clone()
    }

    fn clear_traces(&self) {
        let mut traces = self.traces.write().unwrap();
        traces.clear();
        debug!("Cleared all traces");
    }
}

/// TracingMiddleware is a middleware that adds tracing to a HybridPipe.
pub struct TracingMiddleware {
    /// Inner router
    inner: Arc<dyn HybridPipe>,
    /// Tracer
    tracer: Arc<dyn Tracer>,
    /// Protocol name
    protocol: String,
}

impl TracingMiddleware {
    /// Create a new tracing middleware.
    pub fn new(inner: Arc<dyn HybridPipe>, tracer: Arc<dyn Tracer>, protocol: String) -> Self {
        Self {
            inner,
            tracer,
            protocol,
        }
    }

    /// Create a new tracing middleware with a broker type.
    pub fn new_with_broker_type(
        inner: Arc<dyn HybridPipe>,
        tracer: Arc<dyn Tracer>,
        broker_type: BrokerType,
    ) -> Self {
        Self::new(inner, tracer, format!("{}", broker_type))
    }
}

#[async_trait]
impl HybridPipe for TracingMiddleware {
    async fn connect(&self) -> Result<()> {
        let trace = self.tracer.start_trace(&self.protocol, "connect");
        let result = self.inner.connect().await;
        match &result {
            Ok(_) => {
                self.tracer.end_trace(trace, TraceStatus::Success, None);
            }
            Err(e) => {
                self.tracer
                    .end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
            }
        }
        result
    }

    async fn disconnect(&self) -> Result<()> {
        let trace = self.tracer.start_trace(&self.protocol, "disconnect");
        let result = self.inner.disconnect().await;
        match &result {
            Ok(_) => {
                self.tracer.end_trace(trace, TraceStatus::Success, None);
            }
            Err(e) => {
                self.tracer
                    .end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
            }
        }
        result
    }

    async fn dispatch(&self, pipe: &str, data: Box<dyn Any + Send + Sync>) -> Result<()> {
        let trace = self.tracer.start_trace(&self.protocol, pipe);
        let result = self.inner.dispatch(pipe, data).await;
        match &result {
            Ok(_) => {
                self.tracer.end_trace(trace, TraceStatus::Success, None);
            }
            Err(e) => {
                self.tracer
                    .end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
            }
        }
        result
    }

    async fn dispatch_with_context(
        &self,
        ctx: &context::Context,
        pipe: &str,
        data: Box<dyn Any + Send + Sync>,
    ) -> Result<()> {
        let trace = self.tracer.start_trace(&self.protocol, pipe);
        let result = self.inner.dispatch_with_context(ctx, pipe, data).await;
        match &result {
            Ok(_) => {
                self.tracer.end_trace(trace, TraceStatus::Success, None);
            }
            Err(e) => {
                self.tracer
                    .end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
            }
        }
        result
    }

    async fn subscribe(&self, pipe: &str, callback: Process) -> Result<()> {
        // Create a wrapper that adds tracing
        let tracer = self.tracer.clone();
        let protocol = self.protocol.clone();
        let pipe_name = pipe.to_string();

        let tracing_callback: Process = Arc::new(move |data| {
            let tracer = tracer.clone();
            let protocol = protocol.clone();
            let pipe_name = pipe_name.clone();
            let callback = callback.clone();

            // Process is now a synchronous function, not async
            let trace = tracer.start_trace(&protocol, &pipe_name);
            let result = callback(data);
            match &result {
                Ok(_) => {
                    tracer.end_trace(trace, TraceStatus::Success, None);
                }
                Err(e) => {
                    tracer.end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
                }
            }
            result
        });

        // Subscribe with the wrapper
        self.inner.subscribe(pipe, tracing_callback).await
    }

    async fn subscribe_with_context(
        &self,
        ctx: &context::Context,
        pipe: &str,
        callback: Process,
    ) -> Result<()> {
        // Create a wrapper that adds tracing
        let tracer = self.tracer.clone();
        let protocol = self.protocol.clone();
        let pipe_name = pipe.to_string();
        let ctx_clone = ctx.clone();

        let tracing_callback: Process = Arc::new(move |data| {
            let tracer = tracer.clone();
            let protocol = protocol.clone();
            let pipe_name = pipe_name.clone();
            let callback = callback.clone();
            let ctx = ctx_clone.clone();

            // Check if the context is cancelled
            if ctx.is_cancelled() {
                return Err(Error::ContextCancelled);
            }

            let trace = tracer.start_trace(&protocol, &pipe_name);
            let result = callback(data);
            match &result {
                Ok(_) => {
                    tracer.end_trace(trace, TraceStatus::Success, None);
                }
                Err(e) => {
                    tracer.end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
                }
            }
            result
        });

        // Subscribe with the wrapper
        self.inner.subscribe_with_context(ctx, pipe, tracing_callback).await
    }

    async fn remove(&self, pipe: &str) -> Result<()> {
        self.unsubscribe(pipe).await
    }

    async fn unsubscribe(&self, pipe: &str) -> Result<()> {
        let trace = self.tracer.start_trace(&self.protocol, pipe);
        let result = self.inner.unsubscribe(pipe).await;
        match &result {
            Ok(_) => {
                self.tracer.end_trace(trace, TraceStatus::Success, None);
            }
            Err(e) => {
                self.tracer
                    .end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
            }
        }
        result
    }

    async fn close(&self) -> Result<()> {
        let trace = self.tracer.start_trace(&self.protocol, "close");
        let result = self.inner.close().await;
        match &result {
            Ok(_) => {
                self.tracer.end_trace(trace, TraceStatus::Success, None);
            }
            Err(e) => {
                self.tracer
                    .end_trace(trace, TraceStatus::Error, Some(format!("{}", e)));
            }
        }
        result
    }

    fn is_connected(&self) -> bool {
        self.inner.is_connected()
    }
}
