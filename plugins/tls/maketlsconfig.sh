#!/bin/bash

# Common Definitions
PASSWORD=this4now
VALIDITY=3650
SERVER_HOSTNAME=Anand-iMac

# Definitions for Kafka
KAPPEND=HYBRID.KAFKA.HOST

# Definitions for NATS
NAPPEND=HYBRID.NATS.HOST


# CERT GENERATION START - KAFKA
# Step 1 - Generate CA cert
echo ""
echo "############## FOR KAFKA ##"
echo "Will Create key pair for Root CA, to be used for signing other certs"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
openssl req -new -x509 -keyout $KAPPEND-CA-Key.pem -out $KAPPEND-CA-Cert.pem -days $VALIDITY

# Generate server keystore with RSA private key
echo ""
echo "############## FOR KAFKA ##"
echo "Will create jks with server private key in RSA format"
echo "***** When asked for first and last name"
echo "***** Make sure to provide the hostname client uses for connecting"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
keytool -genkey -alias $SERVER_HOSTNAME -keyalg RSA -keystore $SERVER_HOSTNAME-server.keystore.jks -keysize 2048 -validity $VALIDITY -storepass $PASSWORD

# Generate cert signing request
echo ""
echo "############## FOR KAFKA ##"
echo "Will generate server's cert signing request"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
keytool -keystore $SERVER_HOSTNAME-server.keystore.jks -alias $SERVER_HOSTNAME -certreq -file $SERVER_HOSTNAME-server.csr  -storepass $PASSWORD

# Sign server certificate with self-signed CA from Step 1
echo ""
echo "############## FOR KAFKA ##"
echo "Will sign server certificate"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
openssl x509 -req -CA rootca.cert.pem -CAkey $KAPPEND-CA-Key.pem -in $SERVER_HOSTNAME-server.csr -out $SERVER_HOSTNAME-server.cert.pem -days 365 -CAcreateserial -passin pass:$PASSWORD

# Import RootCA cert from step 1 into server's keystore
echo ""
echo "############## FOR KAFKA ##"
echo "Will import RootCA into server's keystore"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
keytool -keystore $SERVER_HOSTNAME-server.keystore.jks -alias CARoot -import -file $KAPPEND-CA-Cert.pem -storepass $PASSWORD


# Import signed server cert into server's keystore
echo ""
echo "############## FOR KAFKA ##"
echo "Will import signed server cert into server's keystore"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
keytool -keystore $SERVER_HOSTNAME-server.keystore.jks -alias $SERVER_HOSTNAME -import -file $SERVER_HOSTNAME-server.cert.pem -storepass $PASSWORD

# Import RootCA cert into server truststore
echo ""
echo "############## FOR KAFKA ##"
echo "Will import RootCA cert into server's truststore"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
keytool -keystore kafka.server.truststore.jks -alias CARoot -import -file $KAPPEND-CA-Cert.pem -storepass $PASSWORD

### Create client cert
# Create client RSA private key and certificate request
echo ""
echo "############## FOR KAFKA ##"
echo "Will create client RSA private key and certificate request"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
openssl req -nodes -new -keyout $KAPPEND-Client-Key.pem -out $KAPPEND-Client.csr -days $VALIDITY

# Sign client certificate with CA cert from step 1
echo ""
echo "############## FOR KAFKA ##"
echo "Will sign client certificate with CA cert and key from step 1"
echo "##############"
read -n1 -rsp $'Press any key to continue or Ctrl+C to exit...\n'
openssl x509 -req -CA $KAPPEND-CA-Cert.pem -CAkey $KAPPEND-CA-Key.pem -in $KAPPEND-Client.csr -out $KAPPEND-Client.Cert.pem -days 3650 -CAcreateserial

echo ""
echo "############## FOR KAFKA ##"
echo "Following files were generated - For Kafka Communicator Module"
echo "###########################"

echo "Server java keystore: $SERVER_HOSTNAME-server.keystore.jks"
echo "Server java truststore: kafka.server.truststore.jks"
echo "Signed Client cert: mytestclient.cert.pem"
echo "Client RSA private key: mytestclient.key.pem"
echo "Client PEM truststore: rootca.cert.pem"

