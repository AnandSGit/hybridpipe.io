# HybridPipe Serialization Utilities

This document describes common serialization utilities and functions that should be implemented across all language versions of HybridPipe.

## Message Encoding and Decoding

All HybridPipe implementations should provide the following core functions for message encoding and decoding:

### Encode

```
Encode(data, options) -> bytes
```

Encodes data into a serialized byte array using the specified serialization options.

**Parameters:**
- `data`: The data to encode (object, array, string, etc.)
- `options`: Serialization options (format, compression, etc.)

**Returns:**
- Byte array containing the serialized data with HybridPipe header

### Decode

```
Decode(bytes, options) -> data
```

Decodes a serialized byte array into its original data format.

**Parameters:**
- `bytes`: The serialized byte array
- `options`: Serialization options (expected format, etc.)

**Returns:**
- The decoded data in its original format

## Format-Specific Functions

Each serialization format should have specific encode and decode functions:

### JSON

```
EncodeJSON(data, options) -> bytes
DecodeJSON(bytes, options) -> data
```

### Protocol Buffers

```
EncodeProtobuf(data, options) -> bytes
DecodeProtobuf(bytes, options) -> data
```

### MessagePack

```
EncodeMsgPack(data, options) -> bytes
DecodeMsgPack(bytes, options) -> data
```

### CBOR

```
EncodeCBOR(data, options) -> bytes
DecodeCBOR(bytes, options) -> data
```

### BSON

```
EncodeBSON(data, options) -> bytes
DecodeBSON(bytes, options) -> data
```

## Compression Utilities

HybridPipe supports compression for large messages. The following compression utilities should be implemented:

### Compress

```
Compress(data) -> bytes
```

Compresses data using the ZLIB algorithm.

**Parameters:**
- `data`: The data to compress

**Returns:**
- Compressed byte array

### Decompress

```
Decompress(bytes) -> data
```

Decompresses data that was compressed using the ZLIB algorithm.

**Parameters:**
- `bytes`: The compressed byte array

**Returns:**
- Decompressed byte array

## Message Header Utilities

Utilities for working with message headers:

### CreateHeader

```
CreateHeader(format, compression, payloadLength) -> header
```

Creates a new message header.

**Parameters:**
- `format`: The serialization format
- `compression`: Whether the payload is compressed
- `payloadLength`: The length of the payload in bytes

**Returns:**
- A new message header

### HeaderToBytes

```
HeaderToBytes(header) -> bytes
```

Converts a message header to its binary representation.

**Parameters:**
- `header`: The message header

**Returns:**
- Binary representation of the header

### BytesToHeader

```
BytesToHeader(bytes) -> header
```

Converts a binary representation of a header to a header object.

**Parameters:**
- `bytes`: Binary representation of the header

**Returns:**
- Message header object

## Cross-Language Compatibility

To ensure compatibility between different language implementations, all implementations should:

1. Use the same binary format for message headers
2. Use the same serialization formats
3. Use the same compression algorithm (ZLIB)
4. Handle endianness consistently (little-endian for multi-byte values)

## Error Handling

All serialization functions should handle errors gracefully and provide meaningful error messages. Common error cases include:

- Invalid input data
- Unsupported serialization format
- Compression/decompression errors
- Invalid message headers
- Insufficient buffer size

## Performance Considerations

Serialization and deserialization are performance-critical operations in a messaging system. Implementations should:

1. Minimize memory allocations
2. Use efficient algorithms
3. Consider buffer pooling for frequent operations
4. Provide benchmarking tools for performance testing

## Testing

Each implementation should include comprehensive tests for:

1. Encoding and decoding all supported data types
2. Handling edge cases (empty data, very large data, etc.)
3. Cross-language compatibility
4. Error handling
5. Performance benchmarks
