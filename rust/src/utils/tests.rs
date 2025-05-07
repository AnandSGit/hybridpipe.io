#[cfg(test)]
mod tests {
    use crate::utils::{
        current_timestamp_ms, current_timestamp_sec, format_bytes, format_duration, generate_id,
        truncate_string,
    };
    use std::time::Duration;

    #[test]
    fn test_generate_id() {
        // Generate two IDs and ensure they are different
        let id1 = generate_id();
        let id2 = generate_id();

        assert_ne!(id1, id2, "Generated IDs should be unique");

        // Check that the IDs are valid UUIDs
        assert_eq!(id1.len(), 36, "ID should be 36 characters long");
        assert_eq!(id2.len(), 36, "ID should be 36 characters long");

        // Check that the IDs contain hyphens in the right places
        assert_eq!(
            id1.chars().nth(8).unwrap(),
            '-',
            "ID should have a hyphen at position 8"
        );
        assert_eq!(
            id1.chars().nth(13).unwrap(),
            '-',
            "ID should have a hyphen at position 13"
        );
        assert_eq!(
            id1.chars().nth(18).unwrap(),
            '-',
            "ID should have a hyphen at position 18"
        );
        assert_eq!(
            id1.chars().nth(23).unwrap(),
            '-',
            "ID should have a hyphen at position 23"
        );
    }

    #[test]
    fn test_current_timestamp_ms() {
        // Get the current timestamp
        let timestamp = current_timestamp_ms();

        // Check that it's a reasonable value (greater than 0)
        assert!(timestamp > 0, "Timestamp should be greater than 0");

        // Check that it's a reasonable value (not too far in the future)
        // This test assumes the function is called within a reasonable time frame
        assert!(
            timestamp < 2000000000000,
            "Timestamp should not be too far in the future"
        );
    }

    #[test]
    fn test_current_timestamp_sec() {
        // Get the current timestamp
        let timestamp = current_timestamp_sec();

        // Check that it's a reasonable value (greater than 0)
        assert!(timestamp > 0, "Timestamp should be greater than 0");

        // Check that it's a reasonable value (not too far in the future)
        // This test assumes the function is called within a reasonable time frame
        assert!(
            timestamp < 2000000000,
            "Timestamp should not be too far in the future"
        );
    }

    #[test]
    fn test_format_duration() {
        // Test with hours
        let duration = Duration::from_secs(3661); // 1h 1m 1s
        let formatted = format_duration(duration);
        assert_eq!(
            formatted, "1h 1m 1s 0ms",
            "Duration with hours should be formatted correctly"
        );

        // Test with minutes
        let duration = Duration::from_secs(61); // 1m 1s
        let formatted = format_duration(duration);
        assert_eq!(
            formatted, "1m 1s 0ms",
            "Duration with minutes should be formatted correctly"
        );

        // Test with seconds
        let duration = Duration::from_secs(1); // 1s
        let formatted = format_duration(duration);
        assert_eq!(
            formatted, "1s 0ms",
            "Duration with seconds should be formatted correctly"
        );

        // Test with milliseconds
        let duration = Duration::from_millis(500); // 500ms
        let formatted = format_duration(duration);
        assert_eq!(
            formatted, "500ms",
            "Duration with milliseconds should be formatted correctly"
        );

        // Test with mixed values
        let duration = Duration::from_millis(3661500); // 1h 1m 1s 500ms
        let formatted = format_duration(duration);
        assert_eq!(
            formatted, "1h 1m 1s 500ms",
            "Mixed duration should be formatted correctly"
        );
    }

    #[test]
    fn test_truncate_string() {
        // Test with a string shorter than the max length
        let s = "Hello, world!";
        let truncated = truncate_string(s, 20);
        assert_eq!(
            truncated, s,
            "String shorter than max length should not be truncated"
        );

        // Test with a string equal to the max length
        let s = "Hello, world!";
        let truncated = truncate_string(s, 13);
        assert_eq!(
            truncated, s,
            "String equal to max length should not be truncated"
        );

        // Test with a string longer than the max length
        let s = "Hello, world!";
        let truncated = truncate_string(s, 10);
        assert_eq!(
            truncated, "Hello, ...",
            "String longer than max length should be truncated"
        );

        // Test with a very short max length
        let s = "Hello, world!";
        let truncated = truncate_string(s, 5);
        assert_eq!(
            truncated, "He...",
            "String with very short max length should be truncated"
        );

        // Test with an empty string
        let s = "";
        let truncated = truncate_string(s, 10);
        assert_eq!(truncated, "", "Empty string should remain empty");
    }

    #[test]
    fn test_format_bytes() {
        // Test with bytes
        let bytes = 500u64;
        let formatted = format_bytes(bytes);
        assert_eq!(
            formatted, "500 bytes",
            "Bytes should be formatted correctly"
        );

        // Test with kilobytes
        let bytes = 1500u64;
        let formatted = format_bytes(bytes);
        assert_eq!(
            formatted, "1.46 KB",
            "Kilobytes should be formatted correctly"
        );

        // Test with megabytes
        let bytes = 1500000u64;
        let formatted = format_bytes(bytes);
        assert_eq!(
            formatted, "1.43 MB",
            "Megabytes should be formatted correctly"
        );

        // Test with gigabytes
        let bytes = 1500000000u64;
        let formatted = format_bytes(bytes);
        assert_eq!(
            formatted, "1.40 GB",
            "Gigabytes should be formatted correctly"
        );

        // Test with zero
        let bytes = 0u64;
        let formatted = format_bytes(bytes);
        assert_eq!(
            formatted, "0 bytes",
            "Zero bytes should be formatted correctly"
        );
    }
}
