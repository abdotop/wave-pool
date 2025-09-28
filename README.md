# Wave Pool

A complete Wave Money API simulator ecosystem written in Go with web frontend and Flutter mobile app. Test your checkout and webhook integrations safely offline, no real API keys required.

## Overview

Wave Pool provides a comprehensive simulation environment that replicates the Wave Money payment flow:

1. **Backend API Server (Go)** - Simulates Wave's checkout API with authentication, session management, and webhooks
2. **Web Developer Portal** - Browser-based interface for managing API keys, webhooks, and viewing transactions  
3. **Mobile Payment App (Flutter)** - Simulates the customer payment experience with QR code scanning

## Features

### Backend API (Go)
- **Complete Wave Checkout API simulation** with all standard endpoints
- **Authentication system** with phone number + PIN for developers
- **API key management** with permissions (CHECKOUT_API, BALANCE_API)
- **Webhook system** with shared secret and signing secret strategies
- **Payment simulation** via QR codes and deep links
- **Session management** with token-based authentication
- **SQLite database** with automatic migrations

### Web Portal  
- **Developer dashboard** with transaction statistics and history
- **API key management** - create, list, and revoke keys with permissions
- **Webhook management** - configure endpoints with security strategies
- **Payment simulation pages** - QR codes for mobile app integration
- **Modern responsive UI** built with DaisyUI and Tailwind CSS

### Flutter Mobile App
- **QR code scanner** for payment initiation
- **Payment simulation interface** with success/failure buttons
- **Deep link handling** for `wavepool://pay/{session_id}` URLs
- **Session details display** with amount, merchant, and transaction info
- **Native mobile experience** for both Android and iOS

## Quick Start

### 1. Start the Backend Server

```bash
# Install dependencies
go mod tidy

# Run database migrations  
make db-up

# Start the server
go run main.go
```

The server will start on `http://localhost:8080`

### 2. Access the Web Portal

Open `http://localhost:8080` in your browser to access the developer portal.

**First Time Setup:**
1. Enter a phone number (e.g., `+221785626022`) 
2. Enter a 4-digit PIN (e.g., `1234`)
3. Click "Login / Register" to create your developer account

### 3. Create API Keys and Test Payments

1. **Create an API Key:**
   - Go to "API Keys" tab
   - Click "Create API Key"
   - Select permissions (CHECKOUT_API recommended)
   - Copy the generated API key (shown only once)

2. **Create a Checkout Session:**
   ```bash
   curl -X POST http://localhost:8080/v1/checkout/sessions \
     -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "amount": "1000",
       "currency": "XOF", 
       "success_url": "https://example.com/success",
       "error_url": "https://example.com/error"
     }'
   ```

3. **Test Payment Simulation:**
   - Visit the returned `wave_launch_url` in your browser
   - You'll see a QR code and payment details
   - Use the Flutter app to scan the QR code or simulate payment directly

### 4. Mobile App (Optional)

```bash
cd flutter_app
flutter pub get
flutter run
```

## API Documentation

### Authentication Endpoints
- `POST /api/v1/auth/login` - Login/register with phone + PIN
- `GET /api/v1/users/exists?phone_number=...` - Check if user exists

### Portal Endpoints (Session Auth)
- `GET /api/v1/portal/secrets` - List API keys and webhooks  
- `POST /api/v1/portal/secrets` - Create API key
- `POST /api/v1/portal/webhooks` - Create webhook
- `DELETE /api/v1/portal/secrets/{id}` - Revoke secret
- `GET /api/v1/portal/checkout-sessions` - List transactions

### Wave Checkout API Simulation (API Key Auth)
- `POST /v1/checkout/sessions` - Create checkout session
- `GET /v1/checkout/sessions/{id}` - Get session details
- `GET /v1/checkout/sessions?transaction_id=...` - Get by transaction ID
- `GET /v1/checkout/sessions/search?client_reference=...` - Search sessions
- `POST /v1/checkout/sessions/{id}/expire` - Expire session
- `POST /v1/checkout/sessions/{id}/refund` - Refund session

### Payment Simulation
- `GET /pay/{session_id}` - Payment page with QR code
- `POST /pay/{session_id}` - Submit payment simulation result

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Web Portal    │    │   Go Backend     │    │  Flutter App    │
│   (DaisyUI)     │◄──►│   (SQLite)       │◄──►│  (QR Scanner)   │
│                 │    │                  │    │                 │  
│ • Dashboard     │    │ • Wave API       │    │ • Deep Links    │
│ • API Keys      │    │ • Webhooks       │    │ • Payment UI    │
│ • Webhooks      │    │ • Authentication │    │ • Simulation    │
│ • Transactions  │    │ • Sessions       │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Development Workflow

### Typical Integration Testing Flow:

1. **Developer Setup (Web Portal):**
   - Create developer account
   - Generate API key with CHECKOUT_API permission
   - Configure webhook endpoint (optional)

2. **Create Payment Session (API):**
   ```bash
   curl -X POST http://localhost:8080/v1/checkout/sessions \
     -H "Authorization: Bearer wv_..." \
     -d '{"amount":"5000","currency":"XOF",...}'
   ```

3. **Customer Payment (Mobile/Web):**
   - Visit payment page or scan QR code
   - Simulate payment success/failure
   - Backend updates session status

4. **Webhook Delivery (Future):**
   - Backend triggers webhook on status change
   - Developer receives `checkout.session.completed` event

## Configuration

### Environment Variables
- `PORT` - Server port (default: 8080)
- Database file: `wave-pool.db` (created automatically)

### Database Schema
- `users` - Developer accounts (phone + PIN)
- `sessions` - Authentication sessions  
- `secrets` - API keys and webhook secrets
- `checkout_sessions` - Payment session data

## Testing

```bash
# Run Go tests
go test ./...

# Run Flutter tests  
cd flutter_app && flutter test

# Manual API testing
make test-api  # (if available)
```

## Deployment

### Production Considerations
- Use environment-specific database files
- Configure proper CORS for web portal
- Set up HTTPS with valid certificates
- Implement proper webhook retry logic
- Add rate limiting and request validation

### Docker Support
```bash
docker build -t wave-pool .
docker run -p 8080:8080 wave-pool
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add/update tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

---

**Note:** This is a simulation tool for development and testing. It replicates Wave Money's API behavior for integration testing without requiring actual Wave credentials or processing real payments.