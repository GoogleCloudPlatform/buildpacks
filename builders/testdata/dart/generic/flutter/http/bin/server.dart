import 'dart:convert';
import 'dart:io';

import 'package:namer/namer.dart';
import 'package:shelf/shelf.dart';
import 'package:shelf/shelf_io.dart';
import 'package:shelf_router/shelf_router.dart';
import 'package:shelf_static/shelf_static.dart';

// Configure routes.
final _router =
    Router()
      ..get('/test', _testPass)
      ..get('/api/name', _nameHandler);

Response _testPass(Request req) {
  return Response.ok('PASS');
}

Response _nameHandler(Request req) {
  return Response.ok(json.encode({'name': animal(adjectives: 0, verbs: 1)}));
}

void handleSigTerm(ProcessSignal signal) async {
  if (signal != ProcessSignal.sigterm) return;
  print('sigterm - exiting srever');
  exit(0);
}

void main(List<String> args) async {
  ProcessSignal.sigterm.watch().listen(handleSigTerm);

  // Use any available host or container IP (usually `0.0.0.0`).
  final ip = InternetAddress.anyIPv4;

  final statichandler = createStaticHandler('/workspace/app/build/web');

  final cascadeHandler =
      Cascade() //
          .add(statichandler)
          .add(_router.call)
          .handler;

  // Configure a pipeline that logs requests.
  final handler = Pipeline()
      .addMiddleware(logRequests())
      .addHandler(cascadeHandler);

  // For running in containers, we respect the PORT environment variable.
  final port = int.parse(Platform.environment['PORT'] ?? '8080');
  final server = await serve(handler, ip, port);
  print('Server listening on port ${server.port}');
}
