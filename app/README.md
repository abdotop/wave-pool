# Wave Pool Mobile App

This Flutter application is the mobile client for the Wave Pool simulator. It provides a simple interface for users to authenticate and interact with the backend services.

## Architecture

The application is structured as follows:

- `lib/`: Contains the main application code.
  - `api/`: (Future use) Contains API client code.
  - `auth/`: Contains authentication-related widgets, screens, and services.
    - `widgets/`: Reusable widgets for the authentication flow.
  - `core/`: Contains core application logic, such as routing and secure storage.
  - `dashboard/`: Contains the user dashboard screen.
  - `splash/`: Contains the splash screen.
- `test/`: Contains unit and widget tests.

## Getting Started

1. **Install Flutter:** Make sure you have the Flutter SDK installed.
2. **Install Dependencies:** Run `flutter pub get` in the `app` directory.
3. **Run the App:** Run `flutter run` to start the application on a connected device or simulator.

## Authentication

The application uses a unified authentication screen for both registration and login. The user enters their phone number and a 4-digit PIN. The application then communicates with the backend to authenticate the user and retrieve an access token, which is stored securely on the device.

## Routing

The application uses the `go_router` package for navigation. The routes are defined in `lib/core/router.dart`. The router includes a redirect guard to protect routes that require authentication.