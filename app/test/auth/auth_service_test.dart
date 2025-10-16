import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';
import 'package:wave_pool/auth/auth_service.dart';
import 'package:wave_pool/core/secure_storage_service.dart';

import 'auth_service_test.mocks.dart';

@GenerateMocks([http.Client, SecureStorageService])
void main() {
  late AuthService authService;
  late MockClient client;
  late MockSecureStorageService secureStorageService;

  setUp(() {
    client = MockClient();
    secureStorageService = MockSecureStorageService();
    authService = AuthService(secureStorageService, client);
  });

  group('AuthService', () {
    test('authenticate success', () async {
      when(client.post(
        any,
        headers: anyNamed('headers'),
        body: anyNamed('body'),
      )).thenAnswer((_) async => http.Response('{"access_token": "test_token"}', 200));

      await authService.authenticate('123456789', '1234');

      verify(secureStorageService.saveToken('test_token')).called(1);
    });

    test('authenticate failure', () async {
      when(client.post(
        any,
        headers: anyNamed('headers'),
        body: anyNamed('body'),
      )).thenAnswer((_) async => http.Response('{}', 401));

      expect(() => authService.authenticate('123456789', '1234'), throwsException);
    });

    test('logout', () async {
      await authService.logout();
      verify(secureStorageService.deleteToken()).called(1);
    });
  });
}