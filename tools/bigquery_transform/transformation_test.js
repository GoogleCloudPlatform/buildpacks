/**
 * @fileoverview Unit tests for transformDependencyLog BigQuery UDF.
 */

const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// Load transformation.js into the current global context (mimicking the BigQuery V8 runtime).
const scriptPath = path.join(__dirname, 'transformation.js');
const scriptCode = fs.readFileSync(scriptPath, 'utf8');
vm.runInThisContext(scriptCode);

describe('transformDependencyLog', () => {
  it('returns original message if data is missing or empty', () => {
    const input = {attributes: {foo: 'bar'}};
    const result = transformDependencyLog(input, {});
    expect(result).toBe(input);

    const nullData = {data: null};
    expect(transformDependencyLog(nullData, {})).toBe(nullData);

    const emptyData = {data: ''};
    expect(transformDependencyLog(emptyData, {})).toBe(emptyData);
  });

  it('returns null or undefined if message itself is null or undefined', () => {
    expect(transformDependencyLog(null, {})).toBeNull();
    expect(transformDependencyLog(undefined, {})).toBeUndefined();
  });

  it('transforms standard unencoded textproto message into BigQuery JSON schema', () => {
    const textproto = [
      'runtime: "nodejs"',
      'language: "javascript"',
      'region: "us-central1"',
      'dependencies {',
      '  package_name: "express"',
      '  package_version: "4.18.2"',
      '  explicit_dep: true',
      '  dep_type: "DEPENDENCY_TYPE_NORMAL"',
      '}',
      'dependencies {',
      '  package_name: "lodash"',
      '  package_version: "4.17.21"',
      '  explicit_dep: false',
      '  dep_type: "DEPENDENCY_TYPE_DEV"',
      '}',
    ].join('\n');

    const message = {data: textproto};
    const result = transformDependencyLog(message, {});

    expect(typeof result.data).toBe('string');
    const parsed = JSON.parse(result.data);

    expect(parsed.runtime).toBe('nodejs');
    expect(parsed.language).toBe('javascript');
    expect(parsed.region).toBe('us-central1');
    expect(parsed.dependencies.length).toBe(2);

    expect(parsed.dependencies[0].package_name).toBe('express');
    expect(parsed.dependencies[0].package_version).toBe('4.18.2');
    expect(parsed.dependencies[0].explicit_dep).toBe(true);
    expect(parsed.dependencies[0].dep_type).toBe('DEPENDENCY_TYPE_NORMAL');

    expect(parsed.dependencies[1].package_name).toBe('lodash');
    expect(parsed.dependencies[1].package_version).toBe('4.17.21');
    expect(parsed.dependencies[1].explicit_dep).toBe(false);
    expect(parsed.dependencies[1].dep_type).toBe('DEPENDENCY_TYPE_DEV');
  });

  it('preserves message attributes and other metadata fields on message when transforming data', () => {
    const textproto = 'runtime: "nodejs"\nlanguage: "javascript"';
    const message = {
      data: textproto,
      attributes: {pubsub_message_id: 'abc-123', origin: 'buildpacks'},
      publishTime: '2026-07-13T00:00:00Z',
    };
    const result = transformDependencyLog(message, {});

    expect(result.attributes).toEqual({
      pubsub_message_id: 'abc-123',
      origin: 'buildpacks',
    });
    expect(result.publishTime).toBe('2026-07-13T00:00:00Z');
    const parsed = JSON.parse(result.data);
    expect(parsed.runtime).toBe('nodejs');
    expect(parsed.language).toBe('javascript');
  });

  it('sets top-level scalar string fields to null when missing from textproto', () => {
    const textproto = 'runtime: "ruby"';
    const message = {data: textproto};
    const result = transformDependencyLog(message, {});

    const parsed = JSON.parse(result.data);
    expect(parsed.runtime).toBe('ruby');
    expect(parsed.language).toBeNull();
    expect(parsed.region).toBeNull();
  });

  it('handles all dependency block delimiter styles: {} vs <> with and without preceding colon', () => {
    const textproto = [
      'runtime: "go"',
      'dependencies: {',
      '  package_name: "dep-colon-braces"',
      '}',
      'dependencies {',
      '  package_name: "dep-no-colon-braces"',
      '}',
      'dependencies: <',
      '  package_name: "dep-colon-angles"',
      '>',
      'dependencies <',
      '  package_name: "dep-no-colon-angles"',
      '>',
    ].join('\n');

    const message = {data: textproto};
    const result = transformDependencyLog(message, {});
    const parsed = JSON.parse(result.data);

    expect(parsed.dependencies.length).toBe(4);
    expect(parsed.dependencies[0].package_name).toBe('dep-colon-braces');
    expect(parsed.dependencies[1].package_name).toBe('dep-no-colon-braces');
    expect(parsed.dependencies[2].package_name).toBe('dep-colon-angles');
    expect(parsed.dependencies[3].package_name).toBe('dep-no-colon-angles');
  });

  it('correctly parses explicit_dep values including 0, 1, true, false, case insensitivity, and defaults to false when omitted', () => {
    const textproto = [
      'dependencies { package_name: "pkg-1" explicit_dep: 1 }',
      'dependencies { package_name: "pkg-0" explicit_dep: 0 }',
      'dependencies { package_name: "pkg-true" explicit_dep: True }',
      'dependencies { package_name: "pkg-false" explicit_dep: FALSE }',
      'dependencies { package_name: "pkg-omitted" }',
    ].join('\n');

    const message = {data: textproto};
    const result = transformDependencyLog(message, {});
    const parsed = JSON.parse(result.data);

    expect(parsed.dependencies.length).toBe(5);
    expect(parsed.dependencies[0].explicit_dep).toBe(true);
    expect(parsed.dependencies[1].explicit_dep).toBe(false);
    expect(parsed.dependencies[2].explicit_dep).toBe(true);
    expect(parsed.dependencies[3].explicit_dep).toBe(false);
    expect(parsed.dependencies[4].explicit_dep).toBe(false);
  });

  it('sets individual dependency fields to null when they are not specified inside a block', () => {
    const textproto = 'dependencies { package_name: "express" }';
    const message = {data: textproto};
    const result = transformDependencyLog(message, {});
    const parsed = JSON.parse(result.data);

    expect(parsed.dependencies.length).toBe(1);
    expect(parsed.dependencies[0].package_name).toBe('express');
    expect(parsed.dependencies[0].package_version).toBeNull();
    expect(parsed.dependencies[0].dep_type).toBeNull();
    expect(parsed.dependencies[0].explicit_dep).toBe(false);
  });

  it('decodes base64 string using Buffer before parsing dependencies when atob is undefined', () => {
    const textproto = [
      'runtime: "python"',
      'language: "python"',
      'region: "europe-west1"',
      'dependencies: <',
      '  package_name: "requests"',
      '  package_version: "2.31.0"',
      '  explicit_dep: 1',
      '  dep_type: "DEPENDENCY_TYPE_NORMAL"',
      '>',
    ].join('\n');

    const base64Data = Buffer.from(textproto, 'utf8').toString('base64');
    const message = {data: base64Data};
    const result = transformDependencyLog(message, {});

    const parsed = JSON.parse(result.data);
    expect(parsed.runtime).toBe('python');
    expect(parsed.language).toBe('python');
    expect(parsed.region).toBe('europe-west1');
    expect(parsed.dependencies.length).toBe(1);
    expect(parsed.dependencies[0].package_name).toBe('requests');
    expect(parsed.dependencies[0].package_version).toBe('2.31.0');
    expect(parsed.dependencies[0].explicit_dep).toBe(true);
  });

  it('decodes base64 string using global atob if typeof atob === "function"', () => {
    const textproto = [
      'runtime: "java"',
      'language: "java"',
      'dependencies { package_name: "guava" package_version: "31.1-jre" }',
    ].join('\n');

    const base64Data = Buffer.from(textproto, 'utf8').toString('base64');

    // Temporarily mock global atob to test the atob code path in transformation.js
    const originalAtob = global.atob;
    let atobCalled = false;
    global.atob = (encoded) => {
      atobCalled = true;
      return Buffer.from(encoded, 'base64').toString('utf8');
    };

    try {
      const message = {data: base64Data};
      const result = transformDependencyLog(message, {});
      const parsed = JSON.parse(result.data);

      expect(atobCalled).toBe(true);
      expect(parsed.runtime).toBe('java');
      expect(parsed.dependencies[0].package_name).toBe('guava');
    } finally {
      if (originalAtob !== undefined) {
        global.atob = originalAtob;
      } else {
        delete global.atob;
      }
    }
  });

  it('handles empty dependencies gracefully', () => {
    const textproto = 'runtime: "go"\nlanguage: "go"\nregion: "us-east1"';
    const message = {data: textproto};
    const result = transformDependencyLog(message, {});

    const parsed = JSON.parse(result.data);
    expect(parsed.runtime).toBe('go');
    expect(parsed.language).toBe('go');
    expect(parsed.region).toBe('us-east1');
    expect(parsed.dependencies).toEqual([]);
  });
});

