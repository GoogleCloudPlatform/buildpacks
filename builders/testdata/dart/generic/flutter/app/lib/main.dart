import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Flutter Cloud Buildpacks Sample',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
      ),
      home: const MyHomePage(title: 'Flutter Cloud Buildpacks Sample'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  String name = '';
  SnackBar? snackBar;

  final client = Client();

  @override
  Widget build(BuildContext context) {
    if (snackBar != null) {
      ScaffoldMessenger.of(context).showSnackBar(snackBar!);
      snackBar = null;
    }
    return Scaffold(
      appBar: AppBar(
        backgroundColor: Theme.of(context).colorScheme.inversePrimary,
        title: Text(widget.title),
      ),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            TextFormField(
              readOnly: true,
              textAlign: TextAlign.center,
              decoration: InputDecoration(hintText: name),
            ),
            ElevatedButton(
              style: ElevatedButton.styleFrom(
                textStyle: const TextStyle(fontSize: 20),
              ),
              onPressed: () async {
                try {
                  final response = await client.get(Uri.parse('/api/name'));
                  if (response.statusCode != 200) {
                    throw '${response.statusCode}';
                  }
                  setState(() {
                    final data =
                        json.decode(response.body) as Map<String, Object?>;
                    name = '${data['name'] ?? 'empty name'}';
                  });
                } catch (e) {
                  setState(() {
                    snackBar = SnackBar(content: Text('Ah shoot, error: $e'));
                  });
                }
              },
              child: const Text('Fetch Name'),
            ),
          ],
        ),
      ),
    );
  }
}
