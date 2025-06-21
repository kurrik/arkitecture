/**
 * Tests for CLI watch mode functionality
 */

import fs from 'fs/promises';
import path from 'path';
import { CliApp } from '../../src/cli/index';

// Mock dependencies
jest.mock('chokidar');
jest.mock('fs/promises');

import chokidar from 'chokidar';

describe('CLI Watch Mode', () => {
  let cliApp: CliApp;
  let mockWatcher: {
    on: jest.Mock;
    close: jest.Mock;
  };
  let mockFs: {
    readFile: jest.Mock;
    writeFile: jest.Mock;
  };
  let consoleSpy: {
    log: jest.SpyInstance;
    error: jest.SpyInstance;
  };
  let processExitSpy: jest.SpyInstance;
  let processOnceSpy: jest.SpyInstance;

  beforeEach(() => {
    // Mock chokidar watcher
    mockWatcher = {
      on: jest.fn(),
      close: jest.fn(),
    };
    (chokidar.watch as jest.Mock) = jest.fn().mockReturnValue(mockWatcher);

    // Mock fs promises
    mockFs = {
      readFile: jest.fn(),
      writeFile: jest.fn(),
    };
    (fs.readFile as jest.Mock) = mockFs.readFile;
    (fs.writeFile as jest.Mock) = mockFs.writeFile;

    // Mock console
    consoleSpy = {
      log: jest.spyOn(console, 'log').mockImplementation(),
      error: jest.spyOn(console, 'error').mockImplementation(),
    };

    // Mock process.exit to prevent test termination
    processExitSpy = jest.spyOn(process, 'exit').mockImplementation(() => {
      return undefined as never;
    });

    // Mock process.once to prevent signal handler buildup
    processOnceSpy = jest.spyOn(process, 'once').mockImplementation(() => {
      return process;
    });

    cliApp = new CliApp();
  });

  afterEach(() => {
    jest.resetAllMocks();
    consoleSpy.log.mockRestore();
    consoleSpy.error.mockRestore();
    processExitSpy.mockRestore();
    processOnceSpy.mockRestore();
  });

  describe('Watch mode activation', () => {
    it('should start watch mode when --watch flag is provided', async () => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);

      // Don't actually wait for the promise - just test setup
      const watchPromise = cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);

      // Give it a moment to set up
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(chokidar.watch).toHaveBeenCalledWith('test.ark', {
        persistent: true,
        ignoreInitial: true,
      });

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('🔍 Watching test.ark for changes...')
      );

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('Press Ctrl+C to stop watching')
      );
    });

    it('should process file initially when watch mode starts', async () => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);

      // Start watch mode
      const watchPromise = cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);

      // Give it a moment to process
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(mockFs.readFile).toHaveBeenCalledWith('test.ark', 'utf-8');
      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('Processing initial file...')
      );
    });

    it('should resolve output path when not provided', async () => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);

      const watchPromise = cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);

      await new Promise(resolve => setTimeout(resolve, 10));

      expect(mockFs.writeFile).toHaveBeenCalledWith(
        'test.svg',
        expect.any(String),
        'utf-8'
      );
    });
  });

  describe('File change handling', () => {
    let changeCallback: () => void;
    let addCallback: () => void;
    let unlinkCallback: () => void;
    let errorCallback: (error: Error) => void;

    beforeEach(async () => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);

      // Mock watcher event registration
      mockWatcher.on.mockImplementation((event: string, callback: any) => {
        switch (event) {
          case 'change':
            changeCallback = callback;
            break;
          case 'add':
            addCallback = callback;
            break;
          case 'unlink':
            unlinkCallback = callback;
            break;
          case 'error':
            errorCallback = callback;
            break;
        }
      });

      // Start watch mode
      cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);
      await new Promise(resolve => setTimeout(resolve, 10));
    });

    it('should handle file changes with debouncing', async () => {
      expect(changeCallback).toBeDefined();

      // Trigger change
      changeCallback();

      // Should show change message
      await new Promise(resolve => setTimeout(resolve, 150));

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('File changed, regenerating...')
      );

      expect(mockFs.readFile).toHaveBeenCalledTimes(2); // Initial + change
    });

    it('should handle file deletion', () => {
      expect(unlinkCallback).toBeDefined();

      unlinkCallback();

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('File deleted: test.ark')
      );
    });

    it('should handle file recreation', async () => {
      expect(addCallback).toBeDefined();

      addCallback();

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('File recreated: test.ark')
      );

      // Should process file after recreation
      await new Promise(resolve => setTimeout(resolve, 150));
      expect(mockFs.readFile).toHaveBeenCalledTimes(2); // Initial + recreate
    });

    it('should handle watch errors gracefully', () => {
      expect(errorCallback).toBeDefined();

      const testError = new Error('Watch error test');
      errorCallback(testError);

      expect(consoleSpy.error).toHaveBeenCalledWith(
        expect.stringContaining('Watch error: Watch error test')
      );
    });
  });

  describe('Error handling in watch mode', () => {
    beforeEach(() => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);

      mockWatcher.on.mockImplementation(() => {});
    });

    it('should continue watching after validation errors', async () => {
      // Mock invalid DSL that will cause validation errors
      mockFs.readFile.mockResolvedValue('invalid --> nonexistent');

      cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('⚠ Validation errors (continuing to watch)')
      );

      expect(mockWatcher.close).not.toHaveBeenCalled();
    });

    it('should continue watching after file system errors', async () => {
      // Start watch mode with valid file first
      mockFs.readFile.mockResolvedValueOnce('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);
      
      let changeHandler: (() => void) | undefined;
      mockWatcher.on.mockImplementation((event: string, callback: any) => {
        if (event === 'change') changeHandler = callback;
      });
      
      const watchPromise = cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);
      await new Promise(resolve => setTimeout(resolve, 10));

      // Now make subsequent reads fail to test error handling during watch
      mockFs.readFile.mockRejectedValue(new Error('File system error'));
      
      // Trigger a file change to cause the error
      if (changeHandler) changeHandler();
      
      // Wait for debounced processing
      await new Promise(resolve => setTimeout(resolve, 150));

      // The error message depends on where the error is caught
      expect(consoleSpy.error).toHaveBeenCalledWith(
        expect.stringMatching(/(?:Internal error|✗ Error): File system error/i)
      );

      expect(mockWatcher.close).not.toHaveBeenCalled();
    });
  });

  describe('Watch mode with CLI options', () => {
    beforeEach(() => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);
      mockWatcher.on.mockImplementation(() => {});
    });

    it('should work with --validate-only flag', async () => {
      cliApp.run(['node', 'arkitecture', 'test.ark', '--watch', '--validate-only']);
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(mockFs.readFile).toHaveBeenCalledWith('test.ark', 'utf-8');
      expect(mockFs.writeFile).not.toHaveBeenCalled();
    });

    it('should work with --verbose flag', async () => {
      cliApp.run(['node', 'arkitecture', 'test.ark', '--watch', '--verbose']);
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('Processing: test.ark -> test.svg')
      );
    });

    it('should work with font options', async () => {
      cliApp.run([
        'node', 'arkitecture', 'test.ark', '--watch',
        '--font-size', '16', '--font-family', 'Helvetica'
      ]);
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(mockFs.readFile).toHaveBeenCalledWith('test.ark', 'utf-8');
    });
  });

  describe('Signal handling', () => {
    let sigintCallback: () => void;
    let sigtermCallback: () => void;

    beforeEach(() => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);
      mockWatcher.on.mockImplementation(() => {});

      // Override the global mock for this specific test
      processOnceSpy.mockImplementation((event: any, callback: any) => {
        if (event === 'SIGINT') sigintCallback = callback;
        if (event === 'SIGTERM') sigtermCallback = callback;
        return process;
      });
    });

    it('should handle SIGINT gracefully', async () => {
      cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(sigintCallback).toBeDefined();

      sigintCallback();

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('🛑 Stopping watch mode...')
      );
      expect(mockWatcher.close).toHaveBeenCalled();
      expect(processExitSpy).toHaveBeenCalledWith(0);
    });

    it('should handle SIGTERM gracefully', async () => {
      cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);
      await new Promise(resolve => setTimeout(resolve, 10));

      expect(sigtermCallback).toBeDefined();

      sigtermCallback();

      expect(consoleSpy.log).toHaveBeenCalledWith(
        expect.stringContaining('🛑 Stopping watch mode...')
      );
      expect(mockWatcher.close).toHaveBeenCalled();
      expect(processExitSpy).toHaveBeenCalledWith(0);
    });
  });

  describe('Debouncing behavior', () => {
    let changeCallback: () => void;

    beforeEach(async () => {
      mockFs.readFile.mockResolvedValue('test { label: "Test" }');
      mockFs.writeFile.mockResolvedValue(undefined);

      mockWatcher.on.mockImplementation((event: string, callback: any) => {
        if (event === 'change') changeCallback = callback;
      });

      cliApp.run(['node', 'arkitecture', 'test.ark', '--watch']);
      await new Promise(resolve => setTimeout(resolve, 10));
    });

    it('should debounce rapid file changes', async () => {
      expect(changeCallback).toBeDefined();

      // Trigger multiple rapid changes
      changeCallback();
      changeCallback();
      changeCallback();

      // Should only process once after debounce period
      await new Promise(resolve => setTimeout(resolve, 150));

      // Initial read + 1 debounced change (not 3)
      expect(mockFs.readFile).toHaveBeenCalledTimes(2);
    });
  });
});