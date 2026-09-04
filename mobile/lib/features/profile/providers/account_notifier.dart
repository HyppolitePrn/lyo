import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../../user/services/user_service.dart';

/// Owns the account-deletion call: whether it is in flight, and why it failed.
///
/// A notifier rather than widget state so the ApiClient can be injected the
/// same way every other feature does it — the screen itself never constructs
/// a client.
class AccountNotifier extends ChangeNotifier {
  AccountNotifier({UserService? userService})
      : _users = userService ?? const UserService(ApiClient());

  final UserService _users;

  bool isDeleting = false;
  String? error;

  /// Erases the signed-in account. Returns true when the account is gone, so
  /// the caller knows to end the session and leave the screen.
  ///
  /// Never throws: a failure lands in [error] for the UI to show, because
  /// there is nothing the caller can do with an exception here that it cannot
  /// do with a false.
  Future<bool> deleteAccount(String token) async {
    if (isDeleting) {
      return false;
    }
    isDeleting = true;
    error = null;
    notifyListeners();

    try {
      await _users.deleteAccount(token);
      isDeleting = false;
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      isDeleting = false;
      error = e.message;
      notifyListeners();
      return false;
    }
  }
}
