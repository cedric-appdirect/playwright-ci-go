const net = require('net');
const { chromium, firefox, webkit } = require('@playwright/test');

// Create a TCP client
const client = new net.Socket();

console.log('Arguments:', process.argv);

// Connection details
const host = 'host.testcontainers.internal'; // Replace with the actual host if needed
const port = process.argv[3]; // Replace with the actual port

console.log('Connecting to server:', host, port);

// Connect to the server
client.connect(port, host, () => {
    console.log('Connected to server');

    // Send data to the server
    client.write('Hello from client!');
});

// Receive data from the server
client.on('data', (data) => {
    console.log('Received from server:', data.toString());

    // Close the connection
    client.end();
});

// Handle errors
client.on('error', (err) => {
    console.error('Connection error:', err);
    process.exit(1);
});

(async () => {
    const serverCr = await chromium.launchServer({ proxy: { server: process.argv[2] }, headless: true, port: 1010, wsPath: 'chromium' });
    const serverFx = await firefox.launchServer({ proxy: { server: process.argv[2] }, headless: true, port: 1011, wsPath: 'firefox' });
    const serverWk = await webkit.launchServer({ proxy: { server: process.argv[2] }, headless: true, port: 1012, wsPath: 'webkit' });
    console.log("ready endpoint:", serverCr.wsEndpoint(), ", ", serverFx.wsEndpoint(), ", ", serverWk.wsEndpoint());
})();
