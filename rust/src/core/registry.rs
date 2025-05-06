// Registry module for HybridPipe
//
// This module defines the registry system for HybridPipe protocol implementations.

use std::sync::{Arc, RwLock};
use std::collections::HashMap;
use std::any::{Any, TypeId};
use std::fmt;

use super::{BrokerType, HybridPipe, RouterFactory, BoxedHybridPipe, Error, Result};

// Global registry for router factories
lazy_static::lazy_static! {
    static ref ROUTER_FACTORIES: RwLock<HashMap<BrokerType, RouterFactory>> = RwLock::new(HashMap::new());
    static ref ROUTER_TYPES: RwLock<HashMap<BrokerType, TypeId>> = RwLock::new(HashMap::new());
}

/// Register a router factory for a specific broker type.
pub fn register_router_factory(broker_type: BrokerType, factory: RouterFactory) {
    let mut factories = ROUTER_FACTORIES.write().unwrap();
    factories.insert(broker_type, factory);
}

/// Register a router type for a specific broker type.
pub fn register_router_type<T: HybridPipe + 'static>(broker_type: BrokerType) {
    let mut types = ROUTER_TYPES.write().unwrap();
    types.insert(broker_type, TypeId::of::<T>());
}

/// Deploy a router for a specific broker type.
pub fn deploy_router(broker_type: BrokerType) -> Result<BoxedHybridPipe> {
    // Try the factory-based approach first
    let factories = ROUTER_FACTORIES.read().unwrap();
    if let Some(factory) = factories.get(&broker_type) {
        let router = factory();
        return Ok(router);
    }

    // If no factory is registered, return an error
    Err(Error::BrokerTypeNotRegistered(broker_type.to_string()))
}

/// Get the type ID for a specific broker type.
pub fn get_router_type(broker_type: BrokerType) -> Option<TypeId> {
    let types = ROUTER_TYPES.read().unwrap();
    types.get(&broker_type).copied()
}
