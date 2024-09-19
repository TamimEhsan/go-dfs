# Go-Distributed-FS

Progress so far
- Create a server which listens to port 4000
- Handle incoming message from telnet
- Write and read from file
- Configure server
- Establish network of nodes
- broadcast data to network
- Prepare for recieving and sending file streams
- Save file recieved from network
- Get and serve non local file from network
- Fix streaming for file contents
- Remove peers after closing

## Sequence Diagrams
### Saving a file
```mermaid
sequenceDiagram
    participant Client
    participant Server
    participant Store1 as Server Storage
    participant TL1 as Server Transport Layer
    participant TL2 as Peer Transport Layer
    participant Peer as Peers
    participant Store2 as Peer Storage

    Client->>Server: Upload file
    Server->>Store1: Save file
    activate Server
    Server->>TL1: Send metadata (filename, size)
    TL1->>TL2: Send metadata by TCP
    TL2->>Peer: Send metadata (filename, size)
    activate Peer
    Peer->>TL2: Lock transport layer
    Server->>Peer: Stream file data
    deactivate Server
    Peer->>Store2: Save file
    Peer->>TL2: Unlock transport layer
    deactivate Peer
```
### Get a file through network
```mermaid
sequenceDiagram
    participant Client
    participant Server
    participant Store1 as Server Storage
    participant TL1 as Server Transport Layer
    participant TL2 as Peer Transport Layer
    participant Peer as Peers
    participant Store2 as Peer Storage

    activate Client
    activate Server
    Client->>Server: Request file
    Server->>Store1: Check local storage
    alt File found
        Store1->>Server: Send file
        Server->>Client: Send file
    else File not found
        Store1->>Server: File not found
        Server->>TL1: Broadcast getfile
        TL1->>TL2: Get File
        TL2->>Peer: Peer receives getfile

        activate Peer
        activate Store2
        Peer->>Store2: Get file 
        Store2->>Peer: Retrieve File Stream
        deactivate Store2
        Peer->>TL2: Send metadata (filename, size)
        TL2->>TL1: Send metadata (filename, size)
        TL1->>Server: Send metadata (filename, size)
        Server->>TL1: Lock transport layer
        Peer->>Server: Stream file data
        deactivate Peer
        Server->>TL1: Unlock transport layer
        Server->>Client: Serve file
    end
    deactivate Server
    deactivate Client
```
