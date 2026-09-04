import 'package:flutter_test/flutter_test.dart';

import 'package:mobile/core/api/api_client.dart';

void main() {
  // 503 is also how the API reports timeouts and outages, so only the
  // "… disabled" wording may be read as a feature gate.
  test('a feature gate is recognised', () {
    expect(
      const ApiException(503, 'favorites are disabled').isFeatureDisabled,
      isTrue,
    );
    expect(
      const ApiException(503, 'live streaming is disabled').isFeatureDisabled,
      isTrue,
    );
  });

  test('other 503 causes are not feature gates', () {
    expect(
      const ApiException(503, 'request timeout').isFeatureDisabled,
      isFalse,
    );
  });

  test('a non-503 status is never a feature gate', () {
    expect(
      const ApiException(403, 'forbidden').isFeatureDisabled,
      isFalse,
    );
  });
}
