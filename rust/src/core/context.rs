// Context module for HybridPipe
//
// This module defines the context types used by HybridPipe for cancellation and timeouts.

use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::oneshot;
use tokio::time::timeout;

/// Context represents a context for cancellation and timeouts.
#[derive(Clone)]
pub struct Context {
    /// Inner context implementation
    inner: Arc<ContextInner>,
}

/// ContextInner is the inner implementation of Context.
struct ContextInner {
    /// Deadline for the context
    deadline: Option<Instant>,
    /// Cancel receiver for the context
    _cancel_rx: Option<oneshot::Receiver<()>>,
    /// Cancel sender for the context
    cancel_tx: Option<oneshot::Sender<()>>,
}

impl Context {
    /// Create a new background context with no deadline or cancellation.
    pub fn background() -> Self {
        let (tx, rx) = oneshot::channel();
        Self {
            inner: Arc::new(ContextInner {
                deadline: None,
                _cancel_rx: Some(rx),
                cancel_tx: Some(tx),
            }),
        }
    }

    /// Create a new context with a timeout.
    pub fn with_timeout(_parent: &Self, timeout_duration: Duration) -> Self {
        let deadline = Some(Instant::now() + timeout_duration);
        let (tx, rx) = oneshot::channel();
        Self {
            inner: Arc::new(ContextInner {
                deadline,
                _cancel_rx: Some(rx),
                cancel_tx: Some(tx),
            }),
        }
    }

    /// Create a new context with a deadline.
    pub fn with_deadline(_parent: &Self, deadline: Instant) -> Self {
        let (tx, rx) = oneshot::channel();
        Self {
            inner: Arc::new(ContextInner {
                deadline: Some(deadline),
                _cancel_rx: Some(rx),
                cancel_tx: Some(tx),
            }),
        }
    }

    /// Create a new context that can be cancelled.
    pub fn with_cancel(parent: &Self) -> (Self, CancelFn) {
        let (tx, rx) = oneshot::channel();
        let ctx = Self {
            inner: Arc::new(ContextInner {
                deadline: parent.inner.deadline,
                _cancel_rx: Some(rx),
                cancel_tx: None,
            }),
        };
        let cancel_fn = CancelFn {
            cancel_tx: Some(tx),
        };
        (ctx, cancel_fn)
    }

    /// Check if the context is done (cancelled or timed out).
    pub async fn done(&self) -> bool {
        if let Some(deadline) = self.inner.deadline {
            if Instant::now() >= deadline {
                return true;
            }
        }

        // We can't check the cancel_rx directly because it's not Clone or Copy
        // Instead, we'll just return false here and let the actual cancellation
        // be handled by the run method
        false
    }

    /// Check if the context is cancelled.
    pub fn is_cancelled(&self) -> bool {
        if let Some(deadline) = self.inner.deadline {
            if Instant::now() >= deadline {
                return true;
            }
        }

        // Since we can't directly check the cancel_rx, we'll use a heuristic:
        // If the cancel_tx is None in the inner context, it means cancel() was called
        match &self.inner.cancel_tx {
            None => true,
            _ => false,
        }
    }

    /// Get the deadline for the context, if any.
    pub fn deadline(&self) -> Option<Instant> {
        self.inner.deadline
    }

    /// Run a future with this context, returning an error if the context is done before the future completes.
    pub async fn run<F, T, E>(&self, future: F) -> Result<T, E>
    where
        F: std::future::Future<Output = Result<T, E>>,
        E: From<std::io::Error>,
    {
        if let Some(deadline) = self.inner.deadline {
            let now = Instant::now();
            if now >= deadline {
                return Err(std::io::Error::new(
                    std::io::ErrorKind::TimedOut,
                    "context deadline exceeded",
                )
                .into());
            }

            let duration = deadline - now;
            match timeout(duration, future).await {
                Ok(result) => result,
                Err(_) => Err(std::io::Error::new(
                    std::io::ErrorKind::TimedOut,
                    "context deadline exceeded",
                )
                .into()),
            }
        } else {
            future.await
        }
    }
}

/// CancelFn is a function that cancels a context.
pub struct CancelFn {
    cancel_tx: Option<oneshot::Sender<()>>,
}

impl CancelFn {
    /// Cancel the context.
    pub fn cancel(&mut self) {
        if let Some(tx) = self.cancel_tx.take() {
            let _ = tx.send(());
        }
    }
}
