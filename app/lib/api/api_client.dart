import 'package:http/http.dart' as http;
import 'package:wave_pool/auth/auth_service.dart';
import 'package:wave_pool/core/secure_storage_service.dart';

class ApiClient extends http.BaseClient {
  final http.Client _inner;
  final AuthService _authService;
  final SecureStorageService _secureStorageService;

  ApiClient(this._inner, this._authService, this._secureStorageService);

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final expiresAt = await _secureStorageService.getExpiresAt();
    if (expiresAt != null && DateTime.now().isAfter(expiresAt)) {
      await _authService.refreshToken();
    }

    final accessToken = await _secureStorageService.getAccessToken();
    if (accessToken != null) {
      request.headers['Authorization'] = 'Bearer $accessToken';
    }

    return _inner.send(request);
  }
}