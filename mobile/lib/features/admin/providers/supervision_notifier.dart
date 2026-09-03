import 'package:flutter/foundation.dart';

import '../../../core/api/api_client.dart';
import '../models/supervision_model.dart';
import '../services/supervision_service.dart';

enum SupervisionStatus { idle, loading, ready, error }

class SupervisionNotifier extends ChangeNotifier {
  SupervisionNotifier({SupervisionService? service})
    : _svc = service ?? const SupervisionService(ApiClient());

  final SupervisionService _svc;

  SupervisionStatus status = SupervisionStatus.idle;
  SupervisionSummary? summary;
  List<Incident> incidents = [];
  String? error;

  /// Set while an acknowledge/resolve request is in flight, so the screen can
  /// disable that one row's buttons rather than the whole list.
  String? pendingIncidentId;

  /// True when the API answered 401: the access token expired. That is a
  /// session to renew, not a permission the account lacks, and the screen says
  /// so rather than showing a bare "forbidden".
  bool sessionExpired = false;

  /// Loads the summary and the feed together: an admin opening this screen
  /// wants both, and issuing them in parallel halves the wait.
  Future<void> load(String token) async {
    status = SupervisionStatus.loading;
    error = null;
    sessionExpired = false;
    notifyListeners();

    try {
      final results = await Future.wait([
        _svc.getSummary(token),
        _svc.listIncidents(token: token),
      ]);
      summary = results[0] as SupervisionSummary;
      incidents = results[1] as List<Incident>;
      status = SupervisionStatus.ready;
    } on ApiException catch (e) {
      status = SupervisionStatus.error;
      sessionExpired = e.statusCode == 401;
      error = e.message;
      // A stale summary next to a "your session ended" banner reads as live
      // data. Drop it: showing nothing is honest, showing old numbers is not.
      if (sessionExpired) {
        summary = null;
        incidents = [];
      }
    } catch (_) {
      status = SupervisionStatus.error;
      error = 'Could not load supervision data.';
    }
    notifyListeners();
  }

  Future<bool> acknowledge(String id, String token) =>
      _update(id, 'acknowledge', token);

  Future<bool> resolve(String id, String token) =>
      _update(id, 'resolve', token);

  Future<bool> _update(String id, String action, String token) async {
    pendingIncidentId = id;
    error = null;
    notifyListeners();

    try {
      final updated = await _svc.updateIncident(
        id: id,
        action: action,
        token: token,
      );
      final index = incidents.indexWhere((i) => i.id == id);
      if (index != -1) {
        incidents[index] = updated;
      }
      // The counts move with the status, so refresh them rather than leaving
      // the header contradicting the list below it.
      summary = await _svc.getSummary(token);
      return true;
    } on ApiException catch (e) {
      sessionExpired = e.statusCode == 401;
      error = e.message;
      return false;
    } catch (_) {
      error = 'Could not update the incident.';
      return false;
    } finally {
      pendingIncidentId = null;
      notifyListeners();
    }
  }
}
