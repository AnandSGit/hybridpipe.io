// Tracing module for HybridPipe
//
// This module provides tracing functionality for HybridPipe.

use async_trait::async_trait;
use std::any::Any;
use std::collections::HashMap;
use std::sync::{Arc, Mutex, RwLock};
use std::time::{Duration, SystemTime};
use uuid::Uuid;

use crate::core::{HybridPipe, Result};
use crate::middleware::Middleware;

/// Span represents a single operation in a trace.
#[derive(Debug, Clone)]
pub struct Span {
    /// Unique ID for this span.
    pub id: String,
    /// ID of the trace this span belongs to.
    pub trace_id: String,
    /// ID of the parent span, if any.
    pub parent_id: Option<String>,
    /// Name of the operation.
    pub name: String,
    /// Start time of the span.
    pub start_time: SystemTime,
    /// End time of the span, if completed.
    pub end_time: Option<SystemTime>,
    /// Whether the span was successful.
    pub success: bool,
    /// Additional attributes for the span.
    pub attributes: HashMap<String, String>,
}

impl Span {
    /// Create a new span.
    pub fn new(name: &str, trace_id: &str, parent_id: Option<&str>) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            trace_id: trace_id.to_string(),
            parent_id: parent_id.map(|s| s.to_string()),
            name: name.to_string(),
            start_time: SystemTime::now(),
            end_time: None,
            success: false,
            attributes: HashMap::new(),
        }
    }

    /// End the span.
    pub fn end(&mut self, success: bool) {
        self.end_time = Some(SystemTime::now());
        self.success = success;
    }

    /// Add an attribute to the span.
    pub fn add_attribute(&mut self, key: &str, value: &str) {
        self.attributes.insert(key.to_string(), value.to_string());
    }

    /// Get the duration of the span.
    pub fn duration(&self) -> Option<Duration> {
        self.end_time
            .map(|end| end.duration_since(self.start_time).unwrap_or_default())
    }
}

/// Trace represents a collection of spans that form a single operation.
#[derive(Debug, Clone)]
pub struct Trace {
    /// Unique ID for this trace.
    pub id: String,
    /// Name of the trace.
    pub name: String,
    /// Start time of the trace.
    pub start_time: SystemTime,
    /// End time of the trace, if completed.
    pub end_time: Option<SystemTime>,
    /// Spans that make up this trace.
    pub spans: Vec<Span>,
}

impl Trace {
    /// Create a new trace.
    pub fn new(name: &str) -> Self {
        Self {
            id: Uuid::new_v4().to_string(),
            name: name.to_string(),
            start_time: SystemTime::now(),
            end_time: None,
            spans: Vec::new(),
        }
    }

    /// End the trace.
    pub fn end(&mut self) {
        self.end_time = Some(SystemTime::now());
    }

    /// Add a span to the trace.
    pub fn add_span(&mut self, span: Span) {
        self.spans.push(span);
    }

    /// Get the duration of the trace.
    pub fn duration(&self) -> Option<Duration> {
        self.end_time
            .map(|end| end.duration_since(self.start_time).unwrap_or_default())
    }
}

/// Tracer collects and manages traces.
pub struct Tracer {
    /// Active traces.
    traces: RwLock<HashMap<String, Trace>>,
    /// Current span for each trace.
    current_spans: RwLock<HashMap<String, Span>>,
    /// Service name.
    service_name: String,
}

impl Tracer {
    /// Create a new tracer.
    pub fn new(service_name: &str) -> Self {
        Self {
            traces: RwLock::new(HashMap::new()),
            current_spans: RwLock::new(HashMap::new()),
            service_name: service_name.to_string(),
        }
    }

    /// Start a new trace.
    pub fn start_trace(&self, name: &str) -> String {
        let trace = Trace::new(name);
        let trace_id = trace.id.clone();

        let mut traces = self.traces.write().unwrap();
        traces.insert(trace_id.clone(), trace);

        trace_id
    }

    /// End a trace.
    pub fn end_trace(&self, trace_id: &str) {
        let mut traces = self.traces.write().unwrap();
        if let Some(trace) = traces.get_mut(trace_id) {
            trace.end();
        }
    }

    /// Start a new span.
    pub fn start_span(&self, name: &str, trace_id: &str, parent_id: Option<&str>) -> String {
        let span = Span::new(name, trace_id, parent_id);
        let span_id = span.id.clone();

        // Add the span to the trace
        let mut traces = self.traces.write().unwrap();
        if let Some(trace) = traces.get_mut(trace_id) {
            trace.add_span(span.clone());
        }

        // Set as current span for the trace
        let mut current_spans = self.current_spans.write().unwrap();
        current_spans.insert(trace_id.to_string(), span);

        span_id
    }

    /// End a span.
    pub fn end_span(&self, trace_id: &str, success: bool) {
        // Get the current span for the trace
        let mut current_spans = self.current_spans.write().unwrap();
        if let Some(span) = current_spans.get_mut(trace_id) {
            span.end(success);
        }
    }

    /// Add an attribute to the current span.
    pub fn add_attribute(&self, trace_id: &str, key: &str, value: &str) {
        let mut current_spans = self.current_spans.write().unwrap();
        if let Some(span) = current_spans.get_mut(trace_id) {
            span.add_attribute(key, value);
        }
    }

    /// Get a trace by ID.
    pub fn get_trace(&self, trace_id: &str) -> Option<Trace> {
        let traces = self.traces.read().unwrap();
        traces.get(trace_id).cloned()
    }

    /// Get all traces.
    pub fn get_traces(&self) -> Vec<Trace> {
        let traces = self.traces.read().unwrap();
        traces.values().cloned().collect()
    }
}

/// TracingMiddleware is a middleware that adds tracing to HybridPipe operations.
pub struct TracingMiddleware {
    /// The tracer to use.
    tracer: Arc<Tracer>,
    /// The current trace ID.
    current_trace_id: Mutex<Option<String>>,
}

impl TracingMiddleware {
    /// Create a new tracing middleware.
    pub fn new(tracer: Arc<Tracer>) -> Self {
        Self {
            tracer,
            current_trace_id: Mutex::new(None),
        }
    }

    /// Start a new trace.
    pub fn start_trace(&self, name: &str) -> String {
        let trace_id = self.tracer.start_trace(name);
        let mut current_trace_id = self.current_trace_id.lock().unwrap();
        *current_trace_id = Some(trace_id.clone());
        trace_id
    }

    /// End the current trace.
    pub fn end_trace(&self) {
        let current_trace_id = self.current_trace_id.lock().unwrap();
        if let Some(trace_id) = current_trace_id.as_ref() {
            self.tracer.end_trace(trace_id);
        }
    }
}

#[async_trait]
impl Middleware for TracingMiddleware {
    fn inner(&self) -> &dyn HybridPipe {
        // This middleware doesn't wrap a HybridPipe directly
        panic!("TracingMiddleware doesn't wrap a HybridPipe directly");
    }

    async fn before_dispatch(&self, pipe: &str, _data: &Box<dyn Any + Send + Sync>) -> Result<()> {
        let current_trace_id = self.current_trace_id.lock().unwrap();
        if let Some(trace_id) = current_trace_id.as_ref() {
            let _span_id = self
                .tracer
                .start_span(&format!("dispatch:{}", pipe), trace_id, None);
            self.tracer.add_attribute(trace_id, "pipe", pipe);
        }
        Ok(())
    }

    async fn after_dispatch(
        &self,
        _pipe: &str,
        _data: &Box<dyn Any + Send + Sync>,
        result: &Result<()>,
    ) {
        let current_trace_id = self.current_trace_id.lock().unwrap();
        if let Some(trace_id) = current_trace_id.as_ref() {
            self.tracer.end_span(trace_id, result.is_ok());
        }
    }

    async fn before_process(&self, pipe: &str, data: &[u8]) -> Result<()> {
        let current_trace_id = self.current_trace_id.lock().unwrap();
        if let Some(trace_id) = current_trace_id.as_ref() {
            let _span_id = self
                .tracer
                .start_span(&format!("process:{}", pipe), trace_id, None);
            self.tracer.add_attribute(trace_id, "pipe", pipe);
            self.tracer
                .add_attribute(trace_id, "size", &data.len().to_string());
        }
        Ok(())
    }

    async fn after_process(&self, _pipe: &str, _data: &[u8], result: &Result<()>) {
        let current_trace_id = self.current_trace_id.lock().unwrap();
        if let Some(trace_id) = current_trace_id.as_ref() {
            self.tracer.end_span(trace_id, result.is_ok());
        }
    }
}
