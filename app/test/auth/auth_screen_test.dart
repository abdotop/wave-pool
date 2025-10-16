import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/mockito.dart';
import 'package:wave_pool/auth/auth_screen.dart';
import 'package:wave_pool/auth/auth_service.dart';

class MockAuthService extends Mock implements AuthService {}

void main() {
  late MockAuthService mockAuthService;

  setUp(() {
    mockAuthService = MockAuthService();
  });

  testWidgets('AuthScreen shows phone input first', (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: AuthScreen(
          authService: mockAuthService,
        ),
      ),
    );

    expect(find.text('Welcome to Wave Pool!'), findsOneWidget);
    expect(find.text('Enter your mobile to start'), findsOneWidget);
  });

  testWidgets('AuthScreen navigates to PIN screen', (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: AuthScreen(
          authService: mockAuthService,
        ),
      ),
    );

    // Enter a valid phone number
    await tester.enterText(find.byType(TextField), '771234567');
    await tester.pump();

    // Tap the next button
    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();

    // Verify that the PIN screen is shown
    expect(find.text('Enter your secret code for account'), findsOneWidget);
  });
}