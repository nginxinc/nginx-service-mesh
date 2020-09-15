const http = require('http');
const url = require('url');
const process = require('process');

let paths = process.env.RECEIVE_PATHS;
if (!paths || paths.length === 0) {
  paths = [ '/echo' ]
} else {
  paths = paths.split(',')
}

let port = parseInt(process.env.PORT);
if (!port) {
  console.log("PORT not set, using :8080");
  port = 8080
}

let numCalls = 0;

handleEcho = (req, resp) => {
  resp.writeHead(200);
  req.on('data', (chunk) => {
    resp.write('Served - ' + chunk + '\n');
  });
  req.on('end', () => {
    console.log('Echoing ' + req.method + ' request: ' + numCalls);
    console.log(JSON.stringify(req.headers));
    resp.end();
  });
}

handleError = (req, resp) => {
  console.log('Generating error - 503 Service Unavailable');
  resp.writeHead(503);
  resp.end('SERVICE UNAVAILABLE');
}

let routes = {
  '/error': handleError,
  '/echo': handleEcho
};
for (i = 0; i < paths.length; i++) {
  routes[paths[i].trim()] = handleEcho;
}

let server = http.createServer((req, resp) => {
  let parts = url.parse(req.url);
  let route = routes[parts.pathname];

  if (route) {
    route(req, resp);
  } else {
    console.log('ERROR: not found: ' + req.url);
    resp.writeHeader(404);
    resp.end('Not Found\n');
  }
  numCalls++;
});

console.log('Running on ' + port);
server.listen(port);
