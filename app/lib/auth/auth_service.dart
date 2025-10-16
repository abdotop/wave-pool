import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:wave_pool/core/secure_storage_service.dart';

class AuthService {
  final SecureStorageService _secureStorageService;
  final http.Client _client;
  final String _baseUrl;

  AuthService(this._secureStorageService, this._client, {String? baseUrl})
      : _baseUrl = baseUrl ?? 'http://localhost:8080/api/v1';

  Future<void> authenticate(String phone, String pin) async {
    final response = await _client.post(
      Uri.parse('$_baseUrl/auth'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'phone': '221$phone', 'pin': pin}),
    );

    if (response.statusCode == 201) {
      final data = jsonDecode(response.body);
      await _secureStorageService.saveTokens(
        accessToken: data['access_token'],
        refreshToken: data['refresh_token'],
        expiresIn: data['expires_in'],
      );
    } else {
      throw Exception('Failed to authenticate: ${response.body}');
    }
  }

  Future<void> refreshToken() async {
    final refreshToken = await _secureStorageService.getRefreshToken();
    if (refreshToken == null) {
      throw Exception('No refresh token available');
    }

    final response = await _client.post(
      Uri.parse('$_baseUrl/auth/refresh'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'refresh_token': refreshToken}),
    );

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      await _secureStorageService.saveTokens(
        accessToken: data['access_token'],
        refreshToken: data['refresh_token'],
        expiresIn: data['expires_in'],
      );
    } else {
      await _secureStorageService.deleteAllTokens();
      throw Exception('Failed to refresh token: ${response.body}');
    }
  }

  Future<void> logout() async {
    final refreshToken = await _secureStorageService.getRefreshToken();
    if (refreshToken != null) {
      await _client.post(
        Uri.parse('$_baseUrl/auth/logout'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'refresh_token': refreshToken}),
      );
    }
    await _secureStorageService.deleteAllTokens();
  }
}