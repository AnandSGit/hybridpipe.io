#[cfg(test)]
mod tests {
    use crate::monitoring::{new_message_stats, MessageStats, PipeMetrics};
    use std::thread::sleep;
    use std::time::Duration;

    #[test]
    fn test_message_stats_new() {
        let stats = MessageStats::new();

        // Check that the initial state is empty
        assert!(
            stats.get_dispatch_metrics().is_empty(),
            "Initial dispatch metrics should be empty"
        );
        assert!(
            stats.get_process_metrics().is_empty(),
            "Initial process metrics should be empty"
        );
    }

    #[test]
    fn test_message_stats_default() {
        let stats = MessageStats::default();

        // Check that the initial state is empty
        assert!(
            stats.get_dispatch_metrics().is_empty(),
            "Default dispatch metrics should be empty"
        );
        assert!(
            stats.get_process_metrics().is_empty(),
            "Default process metrics should be empty"
        );
    }

    #[test]
    fn test_new_message_stats() {
        let stats = new_message_stats();

        // Check that the initial state is empty
        assert!(
            stats.get_dispatch_metrics().is_empty(),
            "Initial dispatch metrics should be empty"
        );
        assert!(
            stats.get_process_metrics().is_empty(),
            "Initial process metrics should be empty"
        );
    }

    #[test]
    fn test_record_dispatch() {
        let stats = MessageStats::new();
        let pipe = "test-pipe";

        // Record a dispatch operation
        stats.record_dispatch_start(pipe);

        // Wait a bit to ensure a measurable duration
        sleep(Duration::from_millis(10));

        // Record the end of the dispatch operation (success)
        stats.record_dispatch_end(pipe, true);

        // Check that the metrics were updated
        let metrics = stats.get_dispatch_metrics();
        assert!(!metrics.is_empty(), "Dispatch metrics should not be empty");
        assert!(
            metrics.contains_key(pipe),
            "Dispatch metrics should contain the pipe"
        );

        let pipe_metrics = metrics.get(pipe).unwrap();
        assert_eq!(pipe_metrics.total_count, 1, "Total count should be 1");
        assert_eq!(pipe_metrics.success_count, 1, "Success count should be 1");
        assert_eq!(pipe_metrics.error_count, 0, "Error count should be 0");
        assert!(
            pipe_metrics.total_time > Duration::from_millis(0),
            "Total time should be greater than 0"
        );
        assert!(pipe_metrics.min_time.is_some(), "Min time should be set");
        assert!(pipe_metrics.max_time.is_some(), "Max time should be set");

        // Record another dispatch operation (error)
        stats.record_dispatch_start(pipe);

        // Wait a bit to ensure a measurable duration
        sleep(Duration::from_millis(20));

        // Record the end of the dispatch operation (error)
        stats.record_dispatch_end(pipe, false);

        // Check that the metrics were updated
        let metrics = stats.get_dispatch_metrics();
        let pipe_metrics = metrics.get(pipe).unwrap();
        assert_eq!(pipe_metrics.total_count, 2, "Total count should be 2");
        assert_eq!(pipe_metrics.success_count, 1, "Success count should be 1");
        assert_eq!(pipe_metrics.error_count, 1, "Error count should be 1");
    }

    #[test]
    fn test_record_process() {
        let stats = MessageStats::new();
        let pipe = "test-pipe";
        let size = 100;

        // Record a process operation
        stats.record_process_start(pipe, size);

        // Wait a bit to ensure a measurable duration
        sleep(Duration::from_millis(10));

        // Record the end of the process operation (success)
        stats.record_process_end(pipe, true);

        // Check that the metrics were updated
        let metrics = stats.get_process_metrics();
        assert!(!metrics.is_empty(), "Process metrics should not be empty");
        assert!(
            metrics.contains_key(pipe),
            "Process metrics should contain the pipe"
        );

        let pipe_metrics = metrics.get(pipe).unwrap();
        assert_eq!(pipe_metrics.total_count, 1, "Total count should be 1");
        assert_eq!(pipe_metrics.success_count, 1, "Success count should be 1");
        assert_eq!(pipe_metrics.error_count, 0, "Error count should be 0");
        assert_eq!(
            pipe_metrics.total_size, size as u64,
            "Total size should match"
        );
        assert!(
            pipe_metrics.total_time > Duration::from_millis(0),
            "Total time should be greater than 0"
        );
        assert!(pipe_metrics.min_time.is_some(), "Min time should be set");
        assert!(pipe_metrics.max_time.is_some(), "Max time should be set");

        // Record another process operation (error)
        stats.record_process_start(pipe, size * 2);

        // Wait a bit to ensure a measurable duration
        sleep(Duration::from_millis(20));

        // Record the end of the process operation (error)
        stats.record_process_end(pipe, false);

        // Check that the metrics were updated
        let metrics = stats.get_process_metrics();
        let pipe_metrics = metrics.get(pipe).unwrap();
        assert_eq!(pipe_metrics.total_count, 2, "Total count should be 2");
        assert_eq!(pipe_metrics.success_count, 1, "Success count should be 1");
        assert_eq!(pipe_metrics.error_count, 1, "Error count should be 1");
        assert_eq!(
            pipe_metrics.total_size,
            size as u64 * 3,
            "Total size should be updated"
        );
    }

    #[test]
    fn test_get_pipe_metrics() {
        let stats = MessageStats::new();
        let pipe = "test-pipe";

        // Record a dispatch operation
        stats.record_dispatch_start(pipe);
        stats.record_dispatch_end(pipe, true);

        // Record a process operation
        stats.record_process_start(pipe, 100);
        stats.record_process_end(pipe, true);

        // Get the metrics for the pipe
        let dispatch_metrics = stats.get_pipe_dispatch_metrics(pipe);
        let process_metrics = stats.get_pipe_process_metrics(pipe);

        assert!(dispatch_metrics.is_some(), "Dispatch metrics should exist");
        assert!(process_metrics.is_some(), "Process metrics should exist");

        // Get metrics for a non-existent pipe
        let dispatch_metrics = stats.get_pipe_dispatch_metrics("non-existent");
        let process_metrics = stats.get_pipe_process_metrics("non-existent");

        assert!(
            dispatch_metrics.is_none(),
            "Dispatch metrics should not exist"
        );
        assert!(
            process_metrics.is_none(),
            "Process metrics should not exist"
        );
    }

    #[test]
    fn test_reset() {
        let stats = MessageStats::new();
        let pipe = "test-pipe";

        // Record a dispatch operation
        stats.record_dispatch_start(pipe);
        stats.record_dispatch_end(pipe, true);

        // Record a process operation
        stats.record_process_start(pipe, 100);
        stats.record_process_end(pipe, true);

        // Check that the metrics were updated
        assert!(
            !stats.get_dispatch_metrics().is_empty(),
            "Dispatch metrics should not be empty"
        );
        assert!(
            !stats.get_process_metrics().is_empty(),
            "Process metrics should not be empty"
        );

        // Reset the metrics
        stats.reset();

        // Check that the metrics were reset
        assert!(
            stats.get_dispatch_metrics().is_empty(),
            "Dispatch metrics should be empty after reset"
        );
        assert!(
            stats.get_process_metrics().is_empty(),
            "Process metrics should be empty after reset"
        );
    }

    #[test]
    fn test_pipe_metrics_default() {
        let metrics = PipeMetrics::default();

        assert_eq!(metrics.total_count, 0, "Total count should be 0");
        assert_eq!(metrics.success_count, 0, "Success count should be 0");
        assert_eq!(metrics.error_count, 0, "Error count should be 0");
        assert_eq!(
            metrics.total_time,
            Duration::from_secs(0),
            "Total time should be 0"
        );
        assert!(metrics.min_time.is_none(), "Min time should be None");
        assert!(metrics.max_time.is_none(), "Max time should be None");
        assert_eq!(metrics.total_size, 0, "Total size should be 0");
    }
}
