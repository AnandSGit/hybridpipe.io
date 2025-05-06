use criterion::{black_box, criterion_group, criterion_main, Criterion};
use hybridpipe::{deploy_router, BrokerType, HybridPipe};
use serde::{Serialize, Deserialize};
use std::sync::Arc;
use std::time::Duration;

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

async fn setup_mock() -> Arc<dyn HybridPipe> {
    let router = deploy_router(BrokerType::MOCK).unwrap();
    router.connect().await.unwrap();
    router
}

fn bench_mock_dispatch(c: &mut Criterion) {
    let rt = tokio::runtime::Runtime::new().unwrap();
    
    let router = rt.block_on(setup_mock());
    let message = create_test_message();
    
    let mut group = c.benchmark_group("mock_protocol");
    group.measurement_time(Duration::from_secs(10));
    
    group.bench_function("dispatch", |b| {
        b.to_async(&rt).iter(|| async {
            let msg = black_box(message.clone());
            router.dispatch("test-pipe", Box::new(msg)).await.unwrap();
        });
    });
    
    group.finish();
    
    rt.block_on(async {
        router.disconnect().await.unwrap();
    });
}

criterion_group!(benches, bench_mock_dispatch);
criterion_main!(benches);
