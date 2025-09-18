# Manual Webhook Integration Test

## Setup Test Server

First, run a simple webhook receiver to test our implementation:

```bash
# Run this in a separate terminal to receive webhooks
python3 -c "
import http.server
import socketserver
import json
from urllib.parse import urlparse

class WebhookHandler(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        print(f'=== Webhook Received ===')
        print(f'URL: {self.path}')
        print(f'Headers:')
        for header, value in self.headers.items():
            print(f'  {header}: {value}')
        print(f'Body: {post_data.decode()}')
        print('========================')
        
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'OK')

with socketserver.TCPServer(('', 8081), WebhookHandler) as httpd:
    print('Webhook receiver running on http://localhost:8081')
    httpd.serve_forever()
"
```

## Test Webhook Creation and Sending

1. **Start the Wave Pool API server:**
```bash
./wave-pool
```

2. **Create a user and get session token:**
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "+221701234567", "pin": "1234"}'
```

3. **Create webhook with SIGNING_SECRET strategy (default):**
```bash
curl -X POST http://localhost:8080/v1/webhooks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_SESSION_TOKEN" \
  -d '{
    "url": "http://localhost:8081/webhook",
    "event_types": ["checkout.session.completed", "webhook.test"]
  }'
```

4. **Create webhook with SHARED_SECRET strategy:**
```bash
curl -X POST http://localhost:8080/v1/webhooks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_SESSION_TOKEN" \
  -d '{
    "url": "http://localhost:8081/webhook-shared",
    "event_types": ["checkout.session.completed"],
    "security_strategy": "SHARED_SECRET"
  }'
```

5. **Test webhooks:**
```bash
# Test SIGNING_SECRET webhook
curl -X POST http://localhost:8080/v1/webhooks/WEBHOOK_ID_1/test \
  -H "Authorization: Bearer YOUR_SESSION_TOKEN"

# Test SHARED_SECRET webhook  
curl -X POST http://localhost:8080/v1/webhooks/WEBHOOK_ID_2/test \
  -H "Authorization: Bearer YOUR_SESSION_TOKEN"
```

## Expected Results

### SIGNING_SECRET webhook should receive:
- `Content-Type: application/json`
- `Wave-Signature: t=1639081943,v1=abc123...` (with valid HMAC)
- No `Authorization` header

### SHARED_SECRET webhook should receive:
- `Content-Type: application/json`
- `Authorization: Bearer wave_sn_WHS_...`
- No `Wave-Signature` header

Both should receive the test event JSON body:
```json
{
  "id": "EV_...",
  "type": "webhook.test", 
  "data": {
    "test_message": "This is a test webhook event",
    "timestamp": "2023-..."
  }
}
```