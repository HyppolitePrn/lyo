import 'package:app_links/app_links.dart';
import 'package:go_router/go_router.dart';

/// Listens for `lyo://` deep links and routes them into the app.
class DeepLinkListener {
  DeepLinkListener(this._router);

  final GoRouter _router;
  final _appLinks = AppLinks();

  Future<void> init() async {
    final initial = await _appLinks.getInitialLink();
    if (initial != null) {
      _handle(initial);
    }
    _appLinks.uriLinkStream.listen(_handle);
  }

  void _handle(Uri uri) {
    if (uri.scheme == 'lyo' && uri.host == 'reset-password') {
      final token = uri.queryParameters['token'];
      if (token != null) {
        _router.go('/reset-password?token=$token');
      }
    }
  }
}
