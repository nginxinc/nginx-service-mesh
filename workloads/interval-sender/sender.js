const http = require('http');
const timers = require('timers');
const process = require('process');

host = process.env.HOST;
if (!host) {
  host = 'http://localhost:8080';
}

var options = {};
path = process.env.REQUEST_PATH;
if (!path) {
  path = '/echo';
}
options.path = path;

method = process.env.METHOD;
if (!method) {
  method = 'GET';
}
options.method = method;

headers = process.env.HEADERS;
if (headers && headers.length > 0) {
  options.headers = {};
  headers = headers.split(',');

  for (i = 0; i < headers.length; i++) {
    let header, value;
    [header, value] = headers[i].split(':');

    console.log(header + ' ' + value);

    options.headers[header.trim()] = value.trim();
  }
}

let startDelay = 5;

console.log('Starting in ' + startDelay + ' seconds\n');
const countDown = timers.setInterval(() => {
  startDelay -= 1;
  if (startDelay === 0) {
    clearInterval(countDown)

    timers.setInterval(() => {
      console.log('Sending request: ' + host + options.path);
      console.log('Configured options: ' + JSON.stringify(options));
      let req = http.request(host, options, (resp) => {
        const { statusCode } = resp;

        let data = '';
        resp.on('data', (chunk) => {
          data += chunk;
        });

        resp.on('end', () => {
          if (statusCode >= 400) {
            console.log('Server error - ' + statusCode + '\n');
          } else {
            console.log('Success response - ' + statusCode + ' ' + data + '\n');
          }
        });
      }).on('error', (e) => {
        console.error(`Got error: ${e.message}`);
      });
      req.end();
    }, 2000);
  } else {
    console.log('Starting in ' + startDelay + ' seconds\n');
  }
}, 1000);
