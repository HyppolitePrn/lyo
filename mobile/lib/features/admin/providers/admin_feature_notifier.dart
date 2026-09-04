import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../models/feature_flag_model.dart';
import '../services/admin_feature_service.dart';

enum AdminFeatureStatus { idle, loading, ready, error }

class AdminFeatureNotifier extends ChangeNotifier {
  AdminFeatureNotifier({AdminFeatureService? adminFeatureService})
      : _svc = adminFeatureService ?? const AdminFeatureService(ApiClient());

  final AdminFeatureService _svc;

  AdminFeatureStatus status = AdminFeatureStatus.idle;
  List<FeatureFlag> flags = [];
  String? error;

  Future<void> load(String token) async {
    status = AdminFeatureStatus.loading;
    error = null;
    notifyListeners();
    try {
      flags = await _svc.listFlags(token: token);
      status = AdminFeatureStatus.ready;
      notifyListeners();
    } on ApiException catch (e) {
      status = AdminFeatureStatus.error;
      error = e.message;
      notifyListeners();
    } catch (_) {
      status = AdminFeatureStatus.error;
      error = 'Could not load feature flags.';
      notifyListeners();
    }
  }

  Future<bool> toggle({
    required String name,
    required bool enabled,
    required String token,
  }) async {
    try {
      final updated =
          await _svc.toggleFlag(name: name, enabled: enabled, token: token);
      flags = [
        for (final f in flags) f.name == updated.name ? updated : f,
      ];
      notifyListeners();
      return true;
    } on ApiException catch (e) {
      error = e.message;
      notifyListeners();
      return false;
    } catch (_) {
      error = 'Could not update feature flag.';
      notifyListeners();
      return false;
    }
  }
}
