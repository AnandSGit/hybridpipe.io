#[cfg(test)]
mod tests {
    use serde::{Serialize, Deserialize};
    use bytes::Bytes;

    use crate::serialization::{
        SerializationFormat, SerializationOptions, encode, decode, encode_with_options, decode_with_options,
    };

    #[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
    struct TestMessage {
        id: u32,
        name: String,
        values: Vec<f64>,
        tags: Vec<String>,
        active: bool,
    }

    fn create_test_message() -> TestMessage {
        TestMessage {
            id: 12345,
            name: "Test Message".to_string(),
            values: vec![1.0, 2.0, 3.0, 4.0, 5.0],
            tags: vec![
                "tag1".to_string(),
                "tag2".to_string(),
                "tag3".to_string(),
                "tag4".to_string(),
                "tag5".to_string(),
            ],
            active: true,
        }
    }

    #[test]
    fn test_bincode_serialization() {
        let message = create_test_message();
        
        // Encode with default options (bincode)
        let encoded = encode(&message).expect("Failed to encode message");
        
        // Decode
        let decoded: TestMessage = decode(&encoded).expect("Failed to decode message");
        
        // Check if the decoded message matches the original
        assert_eq!(decoded, message);
    }

    #[test]
    fn test_json_serialization() {
        let message = create_test_message();
        
        // Create JSON serialization options
        let options = SerializationOptions {
            format: SerializationFormat::Json,
            ..Default::default()
        };
        
        // Encode with JSON
        let encoded = encode_with_options(&message, &options).expect("Failed to encode message");
        
        // Decode
        let decoded: TestMessage = decode_with_options(&encoded, &options).expect("Failed to decode message");
        
        // Check if the decoded message matches the original
        assert_eq!(decoded, message);
    }

    #[test]
    fn test_messagepack_serialization() {
        let message = create_test_message();
        
        // Create MessagePack serialization options
        let options = SerializationOptions {
            format: SerializationFormat::MessagePack,
            ..Default::default()
        };
        
        // Encode with MessagePack
        let encoded = encode_with_options(&message, &options).expect("Failed to encode message");
        
        // Decode
        let decoded: TestMessage = decode_with_options(&encoded, &options).expect("Failed to decode message");
        
        // Check if the decoded message matches the original
        assert_eq!(decoded, message);
    }

    #[test]
    fn test_compression() {
        let message = create_test_message();
        
        // Create options with compression
        let options = SerializationOptions {
            compression: true,
            compression_level: 6,
            ..Default::default()
        };
        
        // Encode with compression
        let encoded = encode_with_options(&message, &options).expect("Failed to encode message");
        
        // Decode
        let decoded: TestMessage = decode_with_options(&encoded, &options).expect("Failed to decode message");
        
        // Check if the decoded message matches the original
        assert_eq!(decoded, message);
    }

    #[test]
    fn test_format_detection() {
        let message = create_test_message();
        
        // Encode with different formats
        let bincode_options = SerializationOptions {
            format: SerializationFormat::Bincode,
            ..Default::default()
        };
        let json_options = SerializationOptions {
            format: SerializationFormat::Json,
            ..Default::default()
        };
        let messagepack_options = SerializationOptions {
            format: SerializationFormat::MessagePack,
            ..Default::default()
        };
        
        let bincode_encoded = encode_with_options(&message, &bincode_options).expect("Failed to encode with bincode");
        let json_encoded = encode_with_options(&message, &json_options).expect("Failed to encode with JSON");
        let messagepack_encoded = encode_with_options(&message, &messagepack_options).expect("Failed to encode with MessagePack");
        
        // Decode with format detection
        let bincode_decoded: TestMessage = decode(&bincode_encoded).expect("Failed to decode bincode");
        let json_decoded: TestMessage = decode(&json_encoded).expect("Failed to decode JSON");
        let messagepack_decoded: TestMessage = decode(&messagepack_encoded).expect("Failed to decode MessagePack");
        
        // Check if the decoded messages match the original
        assert_eq!(bincode_decoded, message);
        assert_eq!(json_decoded, message);
        assert_eq!(messagepack_decoded, message);
    }

    #[test]
    fn test_cross_format_compatibility() {
        let message = create_test_message();
        
        // Encode with bincode
        let bincode_options = SerializationOptions {
            format: SerializationFormat::Bincode,
            ..Default::default()
        };
        let encoded = encode_with_options(&message, &bincode_options).expect("Failed to encode with bincode");
        
        // Decode with explicit bincode options
        let decoded: TestMessage = decode_with_options(&encoded, &bincode_options).expect("Failed to decode with bincode options");
        assert_eq!(decoded, message);
        
        // Decode with format detection
        let auto_decoded: TestMessage = decode(&encoded).expect("Failed to decode with auto detection");
        assert_eq!(auto_decoded, message);
    }

    #[test]
    fn test_error_handling() {
        // Create invalid data
        let invalid_data = Bytes::from_static(&[0, 1, 2, 3, 4]);
        
        // Try to decode
        let result: Result<TestMessage, _> = decode(&invalid_data);
        assert!(result.is_err());
    }
}
