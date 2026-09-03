import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

extension BackNavigation on BuildContext {
  /// Pops the current route, or navigates to [fallback] when there is nothing
  /// to pop back to.
  ///
  /// Auth screens are reached both by `push` (from the splash screen, where a
  /// pop works) and by `go` (after a sign-out or from a guest tapping "sign
  /// in", where the stack has been replaced). Without a fallback the back
  /// arrow silently does nothing on the second path, stranding the user on the
  /// login screen with no way to reach "create account" or return to browsing.
  void popOrGo(String fallback) {
    if (canPop()) {
      pop();
    } else {
      go(fallback);
    }
  }
}
