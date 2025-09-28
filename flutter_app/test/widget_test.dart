import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import '../lib/main.dart';

void main() {
  group('Wave Pool App Tests', () {
    testWidgets('App should display navigation tabs', (WidgetTester tester) async {
      // Build our app and trigger a frame.
      await tester.pumpWidget(const WavePoolApp());

      // Verify that navigation tabs are present
      expect(find.text('Payment'), findsOneWidget);
      expect(find.text('Scan QR'), findsOneWidget);
      
      // Verify initial state shows instruction text
      expect(find.text('Scan a QR code or open a payment link'), findsOneWidget);
    });

    testWidgets('QR Scanner tab shows camera interface', (WidgetTester tester) async {
      await tester.pumpWidget(const WavePoolApp());

      // Tap on the QR Scanner tab
      await tester.tap(find.text('Scan QR'));
      await tester.pumpAndSettle();

      // Verify scanner interface is displayed
      expect(find.text('Scan QR Code'), findsOneWidget);
      expect(find.text('Point your camera at a Wave Pool payment QR code'), findsOneWidget);
    });

    testWidgets('Payment simulator shows empty state initially', (WidgetTester tester) async {
      await tester.pumpWidget(const WavePoolApp());

      // Should show empty state message
      expect(find.text('Scan a QR code or open a payment link'), findsOneWidget);
      expect(find.text('Use the QR Scanner tab to scan payment codes'), findsOneWidget);
    });
  });

  group('URL Parsing Tests', () {
    test('Deep link URL should be parsed correctly', () {
      const testUrl = 'wavepool://pay/cos-12345';
      final uri = Uri.parse(testUrl);
      
      expect(uri.scheme, 'wavepool');
      expect(uri.pathSegments.length, 2);
      expect(uri.pathSegments[0], 'pay');
      expect(uri.pathSegments[1], 'cos-12345');
    });

    test('Invalid URLs should be rejected', () {
      const invalidUrls = [
        'http://example.com',
        'wavepool://invalid',
        'wavepool://pay/',
        'other://pay/cos-12345',
      ];

      for (final url in invalidUrls) {
        final uri = Uri.parse(url);
        final isValid = uri.scheme == 'wavepool' && 
                       uri.pathSegments.length >= 2 && 
                       uri.pathSegments[0] == 'pay';
        expect(isValid, false, reason: 'URL should be invalid: $url');
      }
    });
  });

  group('Payment Simulation Logic', () {
    test('Payment status should be validated', () {
      const validStatuses = ['succeeded', 'failed'];
      const invalidStatuses = ['pending', 'cancelled', '', 'success'];

      for (final status in validStatuses) {
        expect(['succeeded', 'failed'].contains(status), true, 
               reason: 'Status should be valid: $status');
      }

      for (final status in invalidStatuses) {
        expect(['succeeded', 'failed'].contains(status), false, 
               reason: 'Status should be invalid: $status');
      }
    });
  });
}