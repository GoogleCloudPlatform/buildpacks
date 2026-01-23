/**
 * @fileoverview Tests for the bytecode cache generation script.
 */

const fs = require('node:fs');
const path = require('node:path');
const Module = require('node:module');

// Mock the functions used from node:module
const mockEnableCompileCache = jasmine.createSpy('enableCompileCache');
const mockFlushCompileCache = jasmine.createSpy('flushCompileCache');
Module.enableCompileCache = mockEnableCompileCache;
Module.flushCompileCache = mockFlushCompileCache;

/**
 * Helper to run the script.
 */
const runScript = () => {
  require('./main.js');
};

describe('Bytecode Cache Generation Script', () => {
  let originalArgv;
  let originalCwd;
  let processExitSpy;
  let consoleErrorSpy;
  let fsExistsSyncSpy;
  let fsRmSyncSpy;
  let fsMkdirSyncSpy;
  let requireSpy;
  const originalModuleLoad = Module._load;

  const entrypoint = 'test_entrypoint.js';
  const cacheDirName = '.test_cache';
  const cachePath = path.join('/tmp', cacheDirName);

  beforeEach(() => {
    // Backup original values
    originalArgv = process.argv;
    originalCwd = process.cwd;

    // Mock process.argv
    process.argv = ['node', 'main.js', entrypoint, cacheDirName];

    // Mock process.cwd
    process.cwd = () => '/tmp';

    // Spy on process.exit
    processExitSpy = spyOn(process, 'exit').and.stub(); // Prevent test runner exit

    // Spy on console.error
    consoleErrorSpy = spyOn(console, 'error');

    // Spy on fs methods
    fsExistsSyncSpy = spyOn(fs, 'existsSync');
    fsRmSyncSpy = spyOn(fs, 'rmSync');
    fsMkdirSyncSpy = spyOn(fs, 'mkdirSync');

    // Spy on Module._load
    requireSpy = spyOn(Module, '_load').and.callFake((request, parent, isMain) => {
      if (request === entrypoint) {
        // Simulate successful require of the entrypoint
        return { exports: {} };
      }
      // For other requires (like node:fs, etc.), use the original loader
      return originalModuleLoad(request, parent, isMain);
    });

    // Reset mock module spies
    mockEnableCompileCache.calls.reset();
    mockFlushCompileCache.calls.reset();
  });

  afterEach(() => {
    // Restore original values
    process.argv = originalArgv;
    process.cwd = originalCwd;

    // Clear require cache for the main script
    delete require.cache[require.resolve('./main.js')];
  });

  it('should generate cache successfully when cache dir does not exist', () => {
    fsExistsSyncSpy.withArgs(cachePath).and.returnValue(false);

    runScript();

    expect(fsExistsSyncSpy).toHaveBeenCalledWith(cachePath);
    expect(fsRmSyncSpy).not.toHaveBeenCalled();
    expect(fsMkdirSyncSpy).toHaveBeenCalledWith(cachePath);
    expect(mockEnableCompileCache).toHaveBeenCalledWith(cachePath);
    // Check that the entrypoint was required by the script
    expect(requireSpy).toHaveBeenCalledWith(entrypoint, jasmine.anything(), false);
    expect(mockFlushCompileCache).toHaveBeenCalled();
    expect(processExitSpy).not.toHaveBeenCalled();
    expect(consoleErrorSpy).not.toHaveBeenCalled();
  });

  it('should populate cache dir if it exists', () => {
    fsExistsSyncSpy.withArgs(cachePath).and.returnValue(true);

    runScript();

    expect(fsExistsSyncSpy).toHaveBeenCalledWith(cachePath);
    expect(mockEnableCompileCache).toHaveBeenCalledWith(cachePath);
    // Check that the entrypoint was required by the script
    expect(requireSpy).toHaveBeenCalledWith(entrypoint, jasmine.anything(), false);
    expect(mockFlushCompileCache).toHaveBeenCalled();
    expect(processExitSpy).not.toHaveBeenCalled();
    expect(consoleErrorSpy).not.toHaveBeenCalled();
  });

  it('should exit with error if entrypoint is not provided', () => {
    process.argv = ['node', 'main.js'];
    runScript();
    expect(consoleErrorSpy).toHaveBeenCalledWith(jasmine.stringMatching(/Application entrypoint not provided/));
    expect(processExitSpy).toHaveBeenCalledWith(1);
  });

  it('should exit with error if cache directory name is not provided', () => {
    process.argv = ['node', 'main.js', entrypoint];
    runScript();
    expect(consoleErrorSpy).toHaveBeenCalledWith(jasmine.stringMatching(/Cache directory name not provided/));
    expect(processExitSpy).toHaveBeenCalledWith(1);
  });

  it('should exit with error if require fails inside the script', () => {
    fsExistsSyncSpy.and.returnValue(false);
    const requireError = new Error('Module not found');

    // Override the default fake for this test to make entrypoint require fail
    requireSpy.and.callFake((request, parent, isMain) => {
      if (request === entrypoint) {
        throw requireError;
      }
      return originalModuleLoad(request, parent, isMain);
    });

    runScript();

    expect(mockEnableCompileCache).toHaveBeenCalledWith(cachePath);
    expect(consoleErrorSpy).toHaveBeenCalledWith(' Warning: Error during cache generation, build will continue without it.', requireError);
    expect(processExitSpy).toHaveBeenCalledWith(1);
    expect(mockFlushCompileCache).not.toHaveBeenCalled();
  });

  it('should exit with error if fs.mkdirSync fails', () => {
    fsExistsSyncSpy.and.returnValue(false);
    const mkdirError = new Error('Permission denied');
    fsMkdirSyncSpy.and.throwError(mkdirError);

    runScript();

    expect(consoleErrorSpy).toHaveBeenCalledWith(' Warning: Error during cache generation, build will continue without it.', mkdirError);
    expect(processExitSpy).toHaveBeenCalledWith(1);
    expect(mockEnableCompileCache).not.toHaveBeenCalled();
  });
});