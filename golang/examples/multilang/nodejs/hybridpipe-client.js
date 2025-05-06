/**
 * HybridPipe Node.js Client
 * 
 * This module provides a Node.js client for the HybridPipe messaging system.
 * It supports multiple serialization formats for interoperability with the Go implementation.
 */

const zlib = require('zlib');
const { Kafka } = require('kafkajs');
const msgpack = require('@msgpack/msgpack');

// Constants for message format
const MESSAGE_MAGIC = 0x4850; // "HP" in ASCII
const CURRENT_VERSION = 1;
const HEADER_SIZE = 9;

// Serialization formats
const SerializationFormat = {
  GOB: 0,     // Go's native format (not supported in Node.js)
  JSON: 1,    // Standard JSON
  PROTOBUF: 2, // Protocol Buffers
  MSGPACK: 3  // MessagePack
};

// Message flags
const MessageFlags = {
  NONE: 0,
  COMPRESSED: 1
};

class HybridPipeError extends Error {
  constructor(message) {
    super(message);
    this.name = 'HybridPipeError';
  }
}

class SerializationError extends HybridPipeError {
  constructor(message) {
    super(message);
    this.name = 'SerializationError';
  }
}

class DeserializationError extends HybridPipeError {
  constructor(message) {
    super(message);
    this.name = 'DeserializationError';
  }
}

class UnsupportedFormatError extends HybridPipeError {
  constructor(message) {
    super(message);
    this.name = 'UnsupportedFormatError';
  }
}

class SerializationOptions {
  constructor({
    format = SerializationFormat.JSON,
    compression = false,
    compressionLevel = 6,
    protobufMessageType = null
  } = {}) {
    this.format = format;
    this.compression = compression;
    this.compressionLevel = compressionLevel;
    this.protobufMessageType = protobufMessageType;
  }
}

class HybridPipeClient {
  /**
   * Initialize the HybridPipe client.
   * 
   * @param {string} brokerType - The type of messaging system to use (e.g., "kafka", "nats").
   * @param {Object} options - Additional options for the client.
   */
  constructor(brokerType, options = {}) {
    this.brokerType = brokerType.toLowerCase();
    this.options = options;
    this.serializationOptions = new SerializationOptions();
    
    // Initialize the appropriate client based on brokerType
    switch (this.brokerType) {
      case 'kafka':
        this._initKafkaClient();
        break;
      case 'nats':
        this._initNatsClient();
        break;
      case 'rabbitmq':
        this._initRabbitmqClient();
        break;
      case 'redis':
        this._initRedisClient();
        break;
      case 'mqtt':
        this._initMqttClient();
        break;
      default:
        throw new Error(`Unsupported broker type: ${brokerType}`);
    }
  }
  
  /**
   * Initialize a Kafka client.
   * @private
   */
  _initKafkaClient() {
    try {
      // Get Kafka configuration
      const brokers = this.options.brokers || ['localhost:9092'];
      const clientId = this.options.clientId || 'hybridpipe-client';
      
      // Create Kafka client
      this.kafka = new Kafka({
        clientId,
        brokers
      });
      
      // Create producer
      this.producer = this.kafka.producer();
      
      // Consumer will be created when subscribing to a topic
      this.consumer = null;
    } catch (error) {
      throw new Error(`Failed to initialize Kafka client: ${error.message}`);
    }
  }
  
  /**
   * Initialize a NATS client.
   * @private
   */
  _initNatsClient() {
    // Implementation details for NATS
  }
  
  /**
   * Initialize a RabbitMQ client.
   * @private
   */
  _initRabbitmqClient() {
    // Implementation details for RabbitMQ
  }
  
  /**
   * Initialize a Redis client.
   * @private
   */
  _initRedisClient() {
    // Implementation details for Redis
  }
  
  /**
   * Initialize an MQTT client.
   * @private
   */
  _initMqttClient() {
    // Implementation details for MQTT
  }
  
  /**
   * Set the serialization format to use.
   * 
   * @param {number} format - The serialization format.
   */
  setSerializationFormat(format) {
    if (format === SerializationFormat.GOB) {
      throw new UnsupportedFormatError('GOB format is not supported in Node.js');
    }
    
    if (format === SerializationFormat.PROTOBUF && !global.protobufjs) {
      throw new UnsupportedFormatError('Protocol Buffers support requires protobufjs package');
    }
    
    this.serializationOptions.format = format;
  }
  
  /**
   * Enable or disable compression for messages.
   * 
   * @param {boolean} enabled - Whether compression is enabled.
   * @param {number} level - Compression level (1-9, higher = better compression but slower).
   */
  enableCompression(enabled = true, level = 6) {
    this.serializationOptions.compression = enabled;
    this.serializationOptions.compressionLevel = level;
  }
  
  /**
   * Serialize data using the configured format.
   * 
   * @param {any} data - The data to serialize.
   * @returns {Buffer} The serialized data.
   */
  serialize(data) {
    try {
      let payload;
      
      // Serialize based on the format
      switch (this.serializationOptions.format) {
        case SerializationFormat.JSON:
          payload = Buffer.from(JSON.stringify(data), 'utf-8');
          break;
        case SerializationFormat.PROTOBUF:
          if (!global.protobufjs) {
            throw new UnsupportedFormatError('Protocol Buffers support requires protobufjs package');
          }
          // This requires the appropriate protobuf message type to be set up
          if (!this.serializationOptions.protobufMessageType) {
            throw new SerializationError('Protocol Buffer message type not specified');
          }
          // Implementation would depend on how protobuf is set up
          throw new Error('Protocol Buffer serialization not fully implemented');
        case SerializationFormat.MSGPACK:
          payload = msgpack.encode(data);
          break;
        default:
          throw new UnsupportedFormatError(`Unsupported serialization format: ${this.serializationOptions.format}`);
      }
      
      // Apply compression if enabled and beneficial
      let flags = MessageFlags.NONE;
      if (this.serializationOptions.compression && payload.length > 64) {
        const compressed = zlib.gzipSync(payload, {
          level: this.serializationOptions.compressionLevel
        });
        
        if (compressed.length < payload.length) {
          payload = compressed;
          flags = MessageFlags.COMPRESSED;
        }
      }
      
      // Create the message header
      const header = Buffer.alloc(HEADER_SIZE);
      header.writeUInt16BE(MESSAGE_MAGIC, 0);
      header.writeUInt8(CURRENT_VERSION, 2);
      header.writeUInt8(this.serializationOptions.format, 3);
      header.writeUInt8(flags, 4);
      header.writeUInt32BE(payload.length, 5);
      
      // Combine header and payload
      return Buffer.concat([header, payload]);
    } catch (error) {
      throw new SerializationError(`Failed to serialize data: ${error.message}`);
    }
  }
  
  /**
   * Deserialize data.
   * 
   * @param {Buffer} data - The serialized data.
   * @returns {any} The deserialized data.
   */
  deserialize(data) {
    try {
      // Check if the data has our header format
      if (data.length < HEADER_SIZE) {
        throw new DeserializationError('Data too short to contain header');
      }
      
      // Read the header
      const magic = data.readUInt16BE(0);
      const version = data.readUInt8(2);
      const format = data.readUInt8(3);
      const flags = data.readUInt8(4);
      const payloadLength = data.readUInt32BE(5);
      
      // Validate the header
      if (magic !== MESSAGE_MAGIC) {
        throw new DeserializationError('Invalid message magic');
      }
      
      // Check version compatibility
      if (version > CURRENT_VERSION) {
        throw new DeserializationError(`Unsupported protocol version: ${version}`);
      }
      
      // Extract the payload
      if (HEADER_SIZE + payloadLength > data.length) {
        throw new DeserializationError('Invalid payload length');
      }
      
      let payload = data.slice(HEADER_SIZE, HEADER_SIZE + payloadLength);
      
      // Decompress if needed
      if (flags & MessageFlags.COMPRESSED) {
        payload = zlib.gunzipSync(payload);
      }
      
      // Deserialize based on the format
      switch (format) {
        case SerializationFormat.JSON:
          return JSON.parse(payload.toString('utf-8'));
        case SerializationFormat.PROTOBUF:
          if (!global.protobufjs) {
            throw new UnsupportedFormatError('Protocol Buffers support requires protobufjs package');
          }
          // Note: The caller needs to parse this with the correct message type
          return payload;
        case SerializationFormat.MSGPACK:
          return msgpack.decode(payload);
        case SerializationFormat.GOB:
          throw new UnsupportedFormatError('GOB format is not supported in Node.js');
        default:
          throw new UnsupportedFormatError(`Unsupported serialization format: ${format}`);
      }
    } catch (error) {
      throw new DeserializationError(`Failed to deserialize data: ${error.message}`);
    }
  }
  
  /**
   * Send a message to the specified pipe.
   * 
   * @param {string} pipe - The pipe (topic, channel, etc.) to send the message to.
   * @param {any} data - The data to send.
   * @returns {Promise<void>}
   */
  async dispatch(pipe, data) {
    // Serialize the data
    const serialized = this.serialize(data);
    
    // Send based on broker type
    switch (this.brokerType) {
      case 'kafka':
        if (!this.producer.isConnected) {
          await this.producer.connect();
        }
        await this.producer.send({
          topic: pipe,
          messages: [{ value: serialized }]
        });
        break;
      case 'nats':
        // NATS implementation
        break;
      case 'rabbitmq':
        // RabbitMQ implementation
        break;
      case 'redis':
        // Redis implementation
        break;
      case 'mqtt':
        // MQTT implementation
        break;
    }
  }
  
  /**
   * Subscribe to messages from the specified pipe.
   * 
   * @param {string} pipe - The pipe (topic, channel, etc.) to subscribe to.
   * @param {Function} handler - The function to call when a message is received.
   * @returns {Promise<void>}
   */
  async accept(pipe, handler) {
    // Subscribe based on broker type
    switch (this.brokerType) {
      case 'kafka':
        // Create consumer if not already created
        if (!this.consumer) {
          const groupId = this.options.groupId || `hybridpipe-${Date.now()}`;
          this.consumer = this.kafka.consumer({ groupId });
          await this.consumer.connect();
        }
        
        // Subscribe to the topic
        await this.consumer.subscribe({ topic: pipe, fromBeginning: true });
        
        // Start consuming
        await this.consumer.run({
          eachMessage: async ({ topic, partition, message }) => {
            try {
              // Deserialize the message
              const data = this.deserialize(message.value);
              // Call the handler
              await handler(data);
            } catch (error) {
              console.error(`Error processing message: ${error.message}`);
            }
          }
        });
        break;
      case 'nats':
        // NATS implementation
        break;
      case 'rabbitmq':
        // RabbitMQ implementation
        break;
      case 'redis':
        // Redis implementation
        break;
      case 'mqtt':
        // MQTT implementation
        break;
    }
  }
  
  /**
   * Close the client and release resources.
   * @returns {Promise<void>}
   */
  async close() {
    switch (this.brokerType) {
      case 'kafka':
        if (this.producer && this.producer.isConnected) {
          await this.producer.disconnect();
        }
        if (this.consumer && this.consumer.isConnected) {
          await this.consumer.disconnect();
        }
        break;
      case 'nats':
        // NATS cleanup
        break;
      case 'rabbitmq':
        // RabbitMQ cleanup
        break;
      case 'redis':
        // Redis cleanup
        break;
      case 'mqtt':
        // MQTT cleanup
        break;
    }
  }
}

module.exports = {
  HybridPipeClient,
  SerializationFormat,
  SerializationOptions,
  HybridPipeError,
  SerializationError,
  DeserializationError,
  UnsupportedFormatError
};

// Example usage
if (require.main === module) {
  (async () => {
    // Create a client
    const client = new HybridPipeClient('kafka', {
      brokers: ['localhost:9092'],
      clientId: 'hybridpipe-example'
    });
    
    // Set serialization format to JSON (most compatible)
    client.setSerializationFormat(SerializationFormat.JSON);
    
    // Enable compression for large messages
    client.enableCompression(true);
    
    try {
      // Subscribe to a topic
      await client.accept('example-topic', (data) => {
        console.log(`Received message: ${JSON.stringify(data, null, 2)}`);
      });
      
      // Send a message
      await client.dispatch('example-topic', {
        message: 'Hello from Node.js!',
        timestamp: Date.now()
      });
      
      console.log('Message sent. Press Ctrl+C to exit.');
    } catch (error) {
      console.error(`Error: ${error.message}`);
    }
    
    // Keep the program running
    process.on('SIGINT', async () => {
      console.log('Shutting down...');
      await client.close();
      process.exit(0);
    });
  })();
}
