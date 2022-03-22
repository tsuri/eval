'use strict';

const PROTO_PATH = __dirname + '/../proto/engine.proto';
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const client = require("./client");
const fs = require('fs');

let packageDefinition = protoLoader.loadSync(
    PROTO_PATH,
    {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true
    }
);

let EmployeeService = grpc.loadPackageDefinition(
    packageDefinition).engine.EngineService;

let generateCredentials = () => {
  return grpc.credentials.createSsl(
      fs.readFileSync(
          '/data/eval/certificates/certificatesChain/grpc-root-ca-and-grpc-server-ca-and-grpc-client-ca-chain.crt'),
      fs.readFileSync('/data/eval/certificates/clientCertificates/grpc-client.key'),
      fs.readFileSync('/data/eval/certificates/clientCertificates/grpc-client.crt')
  );
}

let grpcClient = new EmployeeService(`golang2021.conf42.com:443`,
                                     generateCredentials());

let employeeId;

if (process.argv.length >= 3) {
  employeeId = process.argv[2];
} else {
  employeeId = 1;
}

client.getEmployeeDetails(grpcClient, employeeId);

