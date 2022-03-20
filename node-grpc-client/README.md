First Demo :
---
A basic unary operation between client and server.

Run `node index.js 4`

---
Second Demo :
---
A git submodule way to store protobufs files so that you dont have to maintain multiple versions.

Show `protos` folder and `.gitmodules` file

Add another git repo as a submodule to current repo.

- `git submodule add git@github.com:mahendrabagul/protobufs.git protos`

Update existing git submodule.

- `git submodule update --remote`

---
Third Demo :
---
Running node-grpc-client by passing server certificate chain.

Show `index.js` file and compare current changes with the earlier commit

---
Fourth Demo :
---
mTLS settings in client to connect to server.

Show `index.js` file and compare current changes with the earlier commit

---
Sixth Demo :
---
Access golang-grpc-server on KinD-based kubernetes cluster behind secured nginx ingress from host.

- Show `index.js` file and compare current changes with the earlier commit
---
