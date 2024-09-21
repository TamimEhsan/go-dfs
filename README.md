# Go-Distributed-FS

Go-DFS is a distributed file system written in golang. The system has the capability to store, retrieve and delete files from local store and also from network. The synchronizations are done in the background with gossip protocol and is eventually consistent. So this is a AP system (from CAP theorem).

## Project Details
### Network layer
The system builds on top of TCP transport layer. The session, connections are managed by the system. 

### Storage
The files are stored in the local file system with the help of a store encapsulation. The storage supports 5 main functionalities
- Check existence of a file
- Store a file
- Retrieve metadata of a file
- Retrieve a file
- Delete a file

### Gossip protocol
To decrease the load on a single node, the synchronization between network is done through gossip protocol or epidemic protocol. When a file is uploaded to any single node, it sends the file to a randomly selected number of peers which in turn send to some other peers. This everntually makes the system consistent. Accroding to CAP theorem, the system is not immediately consistent but available and parition tolerant.

![gossip](https://miro.medium.com/v2/resize:fit:960/1*g-2JSkw7LxpKod2sd4Lt4w.gif)

### Peer discovery
Peer discovery in system is also done by gossip. When a new node joins the cluster it needs to join through a known peer already in the cluster. That known peer sends the peer joining information to the rest of the cluster.

## How to run
### Prequisites
Make sure you have golang installed in your device. Download and install golang from [here](https://go.dev/doc/install)

### Installation
At first clone this repository and then download the dependencies for the project
```bash
git clone https://github.com/TamimEhsan/go-dfs.git
cd go-dfs
go install .
```

### Running the nodes
The servers take 1 mandatory argument <host:port> and optional arguments <addr:port> of the peers. The first node in the system doesn't need any peer address cause there aren't any. The subsequent nodes need at least one remote address to join the network
```bash
go run . 127.0.0.1:4001 # run the first node
go run . 127.0.0.1:4002 127.0.0.1:4001 # run the second node
```
You can add as much number of nodes you want
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

## Acknowledgement
The project is built on top of the tutorial by [Anthony](https://github.com/anthdm) and further modification and new features were added. 