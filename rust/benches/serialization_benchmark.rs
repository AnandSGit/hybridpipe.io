use criterion::{black_box, criterion_group, criterion_main, Criterion};
use hybridpipe::serialization::{
    SerializationFormat, SerializationOptions, encode_with_options, decode_with_options,
};
use serde::{Serialize, Deserialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
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

fn bench_bincode_serialization(c: &mut Criterion) {
    let message = create_test_message();
    let options = SerializationOptions {
        format: SerializationFormat::Bincode,
        compression: false,
        ..Default::default()
    };
    
    c.bench_function("bincode_serialize", |b| {
        b.iter(|| {
            let _ = encode_with_options(black_box(&message), black_box(&options));
        })
    });
    
    let encoded = encode_with_options(&message, &options).unwrap();
    
    c.bench_function("bincode_deserialize", |b| {
        b.iter(|| {
            let _: TestMessage = decode_with_options(black_box(&encoded), black_box(&options)).unwrap();
        })
    });
}

fn bench_json_serialization(c: &mut Criterion) {
    let message = create_test_message();
    let options = SerializationOptions {
        format: SerializationFormat::Json,
        compression: false,
        ..Default::default()
    };
    
    c.bench_function("json_serialize", |b| {
        b.iter(|| {
            let _ = encode_with_options(black_box(&message), black_box(&options));
        })
    });
    
    let encoded = encode_with_options(&message, &options).unwrap();
    
    c.bench_function("json_deserialize", |b| {
        b.iter(|| {
            let _: TestMessage = decode_with_options(black_box(&encoded), black_box(&options)).unwrap();
        })
    });
}

fn bench_messagepack_serialization(c: &mut Criterion) {
    let message = create_test_message();
    let options = SerializationOptions {
        format: SerializationFormat::MessagePack,
        compression: false,
        ..Default::default()
    };
    
    c.bench_function("messagepack_serialize", |b| {
        b.iter(|| {
            let _ = encode_with_options(black_box(&message), black_box(&options));
        })
    });
    
    let encoded = encode_with_options(&message, &options).unwrap();
    
    c.bench_function("messagepack_deserialize", |b| {
        b.iter(|| {
            let _: TestMessage = decode_with_options(black_box(&encoded), black_box(&options)).unwrap();
        })
    });
}

fn bench_compression(c: &mut Criterion) {
    let message = create_test_message();
    
    // Bincode with compression
    let options_compressed = SerializationOptions {
        format: SerializationFormat::Bincode,
        compression: true,
        compression_level: 6,
        ..Default::default()
    };
    
    c.bench_function("bincode_serialize_compressed", |b| {
        b.iter(|| {
            let _ = encode_with_options(black_box(&message), black_box(&options_compressed));
        })
    });
    
    let encoded_compressed = encode_with_options(&message, &options_compressed).unwrap();
    
    c.bench_function("bincode_deserialize_compressed", |b| {
        b.iter(|| {
            let _: TestMessage = decode_with_options(black_box(&encoded_compressed), black_box(&options_compressed)).unwrap();
        })
    });
}

criterion_group!(
    benches,
    bench_bincode_serialization,
    bench_json_serialization,
    bench_messagepack_serialization,
    bench_compression,
);
criterion_main!(benches);
