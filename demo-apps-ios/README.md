# SecureNote iOS App

An iOS application implemented with SwiftUI. This is a demo that showcases native login screen implementation and WebView integration with a React web frontend.

**For Japanese documentation, see [README.ja.md](README.ja.md)**

> ⚠️ **Note:** This demo has not been fully tested and actually contains bugs. If you find any issues, you can use DockMCP to have AI investigate them. That itself is a practical use case for DockMCP.
>
> 📝 **Hint:** The feature descriptions (request/response examples, etc.) also contain bugs. Compare with the Web version in [demo-apps/README.md](../demo-apps/README.md) to find differences.

## Architecture

```
┌─────────────────────────────────────────┐
│ SecureNote iOS App (SwiftUI)            │
├─────────────────────────────────────────┤
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ LoginView (Native)              │   │
│  │ - SwiftUI Form                  │   │
│  │ - API Call                      │   │
│  │ - Token Acquisition             │   │
│  └─────────┬───────────────────────┘   │
│            │ (token)                    │
│            ↓                             │
│  ┌─────────────────────────────────┐   │
│  │ MainView (WebView Integration)  │   │
│  │ - WebViewContainer              │   │
│  │ - Token Injection               │   │
│  │ - React Web App Display         │   │
│  └─────────────────────────────────┘   │
│                                         │
│  API: http://api.securenote.test:8000  │
│  Web: http://securenote.test:8000      │
└─────────────────────────────────────────┘
```

## Features

### 1. Native Login (LoginView)

- **SwiftUI UI**: Native iOS design
- **API Integration**: `/api/auth/login` endpoint
- **Error Handling**: User-friendly error messages
- **Loading State**: UI feedback during login

**Request Example:**
```swift
POST http://api.securenote.test:8000/api/auth/login
Content-Type: application/json

{
  "email": "demo@example.com",
  "password": "demo123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "user123",
    "email": "demo@example.com",
    "name": "Demo User"
  }
}
```

### 2. WebView Integration (WebViewContainer)

- **Token Injection**: Injects authentication token into JavaScript
- **localStorage**: `localStorage.setItem('auth_token', token)`
- **React Web App Integration**: Loads `http://securenote.test:8000`
- **Automatic Authentication**: Web app reads token from localStorage

**Token Injection Script:**
```javascript
window.authToken = 'eyJhbGciOiJIUzI1NiIs...';
localStorage.setItem('auth_token', 'eyJhbGciOiJIUzI1NiIs...');
```

### 3. Authentication Management (AuthManager)

- **State Management**: Maintains login state and token
- **API Communication**: Async/await for non-blocking calls
- **Error Management**: Error message handling and display

## Setup

### Prerequisites

- **macOS**: 11 or later
- **Xcode**: 14 or later
- **iOS**: Target 14 or later
- **Backend**: `demo-apps` running

### How to Run

#### 1. Start the Backend

```bash
cd /workspace/demo-apps
docker-compose -f docker-compose.demo.yml up -d
```

**Verify:**
- API running at `http://api.securenote.test:8000/api`
- Web running at `http://securenote.test:8000`

> **Note:** The simulator references the host OS's `/etc/hosts`. Real devices cannot access the host's hosts file, so you need to either set up a DNS server on your Mac and point the device to it, or route traffic through a proxy (such as Charles or Proxyman).

#### 2. Open in Xcode

```bash
cd /workspace/demo-apps-ios
open SecureNote.xcodeproj
```

#### 3. Run in Simulator

```
Cmd + R (or click Play button)
```

#### 4. Login

Use demo account:
- **Email**: `demo@example.com`
- **Password**: `demo123`

## Development Guide

### File Structure

```
SecureNote/
├── App.swift                # Main entry point
├── AuthManager.swift        # Authentication state
├── LoginView.swift          # Login screen
├── MainView.swift           # Main screen
├── WebViewContainer.swift   # WebView integration
└── Info.plist              # App configuration
```

### Key Classes

#### AuthManager
```swift
@Observable
class AuthManager {
    var isLoggedIn: Bool
    var authToken: String

    func login(email: String, password: String) async
    func logout()
}
```

#### WebViewContainer
```swift
struct WebViewContainer: View {
    let authToken: String
    // Injects token and loads React Web app
}
```

### UI Flow

```
App.swift
├─ isLoggedIn = false
│  └─ LoginView
│     └─ submit email/password
│        └─ AuthManager.login()
│           └─ API POST /api/auth/login
│              ├─ success
│              │  └─ isLoggedIn = true
│              │     └─ token saved
│              └─ error
│                 └─ show error message
│
└─ isLoggedIn = true
   └─ MainView
      └─ WebViewContainer(token: authToken)
         └─ inject token
            └─ load http://securenote.test:8000
               └─ React Web app
```

## API Integration

### Login Endpoint

- **URL**: `http://api.securenote.test:8000/api/auth/login`
- **Method**: `POST`
- **Content-Type**: `application/json`

**Parameters:**
```json
{
  "email": "string",
  "password": "string"
}
```

**Response (200 OK):**
```json
{
  "token": "string",
  "user": {
    "id": "string",
    "email": "string",
    "name": "string"
  }
}
```

**Error Response (4xx, 5xx):**
```json
{
  "message": "string"
}
```

## Troubleshooting

### 1. Cannot Connect to API

**Problem**: `Connection refused` error

**Solution:**
```bash
# Check if backend is running
curl http://api.securenote.test:8000/api/health

# Start if not running
cd /workspace/demo-apps
docker-compose -f docker-compose.demo.yml up -d
```

### 2. Cannot Connect to Custom Domain

**Problem**: iOS simulator cannot access custom domain

**Solution**: Verify that the custom domain is configured in the host OS's `/etc/hosts`. The iOS simulator inherits the macOS `/etc/hosts` and DNS settings directly.

### 3. WebView Shows Blank

**Problem**: React web app not displaying

**Solution:**
- Verify Web is running at `http://securenote.test:8000`
- Test directly in browser: `open http://securenote.test:8000`
- Check console: Xcode → Debug → View Debugger

## Cross-Platform Development

This project demonstrates multi-project collaborative development:

```
/workspace/
├── demo-apps/        # API + React Web (Node.js)
│   ├── securenote-api/
│   ├── securenote-web/
│   └── docker-compose.demo.yml
│
└── demo-apps-ios/    # iOS App (Swift)
    ├── SecureNote/
    └── SecureNote.xcodeproj
```

**Benefits with AI Assistant (Claude Code):**

- ✅ View all source code
- ✅ Debug API errors
- ✅ Verify web frontend integration
- ✅ Investigate cross-platform issues

## References

- [SwiftUI Documentation](https://developer.apple.com/documentation/swiftui/)
- [WebKit Documentation](https://developer.apple.com/documentation/webkit/)
- [URLSession Guide](https://developer.apple.com/documentation/foundation/urlsession)

## License

MIT License
