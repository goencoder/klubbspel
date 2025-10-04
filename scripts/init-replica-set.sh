#!/bin/bash
# Initialize MongoDB replica set after root user is created

echo "Waiting for MongoDB to be ready..."
sleep 10

echo "Initializing replica set..."
mongosh -u root -p pingis123 --authenticationDatabase admin <<EOF
try {
  rs.status();
  print("Replica set already initialized");
} catch(e) {
  if (e.codeName === 'NotYetInitialized') {
    rs.initiate({
      _id: 'rs0',
      members: [{ _id: 0, host: 'localhost:27017' }]
    });
    print("Replica set initialized successfully");
  } else {
    print("Error: " + e);
  }
}
EOF
