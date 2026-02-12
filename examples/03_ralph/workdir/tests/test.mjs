import { describe, test, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Read the HTML file
const htmlPath = join(__dirname, '../src/index.html');
const htmlContent = readFileSync(htmlPath, 'utf-8');

// Parse HTML helper
function parseHTML(html) {
  const doctypeMatch = html.match(/<!DOCTYPE\s+html>/i);
  const styleMatch = html.match(/<style>([\s\S]*?)<\/style>/);
  const scriptMatch = html.match(/<script>([\s\S]*?)<\/script>/);

  return {
    hasDoctype: !!doctypeMatch,
    style: styleMatch ? styleMatch[1] : '',
    script: scriptMatch ? scriptMatch[1] : '',
    html: html
  };
}

const parsed = parseHTML(htmlContent);

// ==========================================
// HTML STRUCTURE TESTS
// ==========================================

describe('HTML Structure Tests', () => {
  test('has DOCTYPE declaration', () => {
    assert.strictEqual(parsed.hasDoctype, true, 'HTML should have DOCTYPE');
  });

  test('has proper meta viewport tag', () => {
    assert.match(
      parsed.html,
      /<meta\s+name="viewport"\s+content="width=device-width,\s*initial-scale=1\.0">/,
      'Should have meta viewport tag'
    );
  });

  test('has title "todos"', () => {
    assert.match(parsed.html, /<title>todos<\/title>/, 'Should have title "todos"');
  });

  test('has header with h1 "todos"', () => {
    assert.match(parsed.html, /<h1>todos<\/h1>/, 'Should have h1 with "todos"');
  });

  test('has form with id="addTaskForm"', () => {
    assert.match(
      parsed.html,
      /<form[^>]+id="addTaskForm"/,
      'Should have form with id="addTaskForm"'
    );
  });

  test('has input with id="addTaskInput" and placeholder', () => {
    assert.match(
      parsed.html,
      /<input[^>]+id="addTaskInput"/,
      'Should have input with id="addTaskInput"'
    );
    assert.match(
      parsed.html,
      /placeholder="What needs to be done\?"/,
      'Should have correct placeholder text'
    );
  });

  test('has task list ul with id="taskList"', () => {
    assert.match(
      parsed.html,
      /<ul[^>]+id="taskList"/,
      'Should have ul with id="taskList"'
    );
  });

  test('has footer with filter buttons (All, Active, Completed)', () => {
    assert.match(
      parsed.html,
      /<footer[^>]+id="footer"/,
      'Should have footer with id="footer"'
    );
    assert.match(
      parsed.html,
      /<button[^>]+data-filter="all">All<\/button>/,
      'Should have "All" filter button'
    );
    assert.match(
      parsed.html,
      /<button[^>]+data-filter="active">Active<\/button>/,
      'Should have "Active" filter button'
    );
    assert.match(
      parsed.html,
      /<button[^>]+data-filter="completed">Completed<\/button>/,
      'Should have "Completed" filter button'
    );
  });

  test('has clear completed button', () => {
    assert.match(
      parsed.html,
      /<button[^>]+id="clearCompletedBtn"/,
      'Should have clear completed button'
    );
    assert.match(
      parsed.html,
      />Clear completed<\/button>/,
      'Should have correct button text'
    );
  });

  test('has task count element', () => {
    assert.match(
      parsed.html,
      /<div[^>]+id="taskCount"/,
      'Should have task count element'
    );
  });
});

// ==========================================
// CSS VERIFICATION TESTS
// ==========================================

describe('CSS Verification Tests', () => {
  test('has CSS custom properties', () => {
    assert.match(parsed.style, /--color-primary:/, 'Should have --color-primary');
    assert.match(parsed.style, /--color-bg:/, 'Should have --color-bg');
    assert.match(parsed.style, /--color-white:/, 'Should have --color-white');
    assert.match(parsed.style, /--color-text:/, 'Should have --color-text');
  });

  test('has responsive media queries', () => {
    assert.match(parsed.style, /@media\s*\([^)]*max-width/, 'Should have media queries');
    assert.match(parsed.style, /@media\s*\([^)]*600px/, 'Should have 600px breakpoint');
  });

  test('has .task-item styles', () => {
    assert.match(parsed.style, /\.task-item\s*{/, 'Should have .task-item class');
    assert.match(parsed.style, /\.task-item[^}]*border-bottom/, 'Should have border-bottom');
  });

  test('has .completed class with strikethrough', () => {
    assert.match(
      parsed.style,
      /\.completed[^}]*text-decoration:\s*line-through/,
      'Should have strikethrough for completed tasks'
    );
  });

  test('has .filter-btn.selected styles', () => {
    assert.match(
      parsed.style,
      /\.filter-btn\.selected/,
      'Should have .filter-btn.selected selector'
    );
  });

  test('has min-height: 44px touch targets', () => {
    assert.match(
      parsed.style,
      /min-height:\s*44px/,
      'Should have 44px min-height for touch targets'
    );
  });

  test('has word-wrap/overflow-wrap for long text', () => {
    const hasWordWrap = /word-wrap:\s*break-word/.test(parsed.style);
    const hasOverflowWrap = /overflow-wrap:\s*break-word/.test(parsed.style);
    assert.ok(
      hasWordWrap || hasOverflowWrap,
      'Should have word-wrap or overflow-wrap'
    );
  });

  test('has :focus-visible styles for accessibility', () => {
    assert.match(
      parsed.style,
      /:focus-visible/,
      'Should have :focus-visible pseudo-class'
    );
    assert.match(
      parsed.style,
      /:focus-visible[^}]*outline/,
      'Should have outline for focus-visible'
    );
  });
});

// ==========================================
// JAVASCRIPT LOGIC TESTS
// ==========================================

describe('JavaScript Logic Tests', () => {
  let storage, TodoStore;
  let mockLocalStorage;

  beforeEach(() => {
    // Create mock localStorage
    mockLocalStorage = {
      data: {},
      getItem(key) {
        return this.data[key] || null;
      },
      setItem(key, value) {
        this.data[key] = value;
      },
      removeItem(key) {
        delete this.data[key];
      },
      clear() {
        this.data = {};
      }
    };

    // Create a mock global object with localStorage
    const mockGlobal = {
      localStorage: mockLocalStorage,
      console: {
        warn: () => {},
        error: () => {},
        log: () => {}
      }
    };

    // Extract and eval the JavaScript
    let jsCode = parsed.script;

    // Remove the initialization code at the end (last 3 lines)
    // This prevents TodoUI from trying to access document
    jsCode = jsCode.replace(/\/\/ Initialize app[\s\S]*$/, '');

    // Wrap the code to capture the storage and TodoStore
    const wrappedCode = `
      const localStorage = mockGlobal.localStorage;
      const console = mockGlobal.console;
      const document = mockGlobal.document || {};
      ${jsCode}
      return { storage, TodoStore };
    `;

    const fn = new Function('mockGlobal', wrappedCode);
    const result = fn(mockGlobal);

    storage = result.storage;
    TodoStore = result.TodoStore;
  });

  afterEach(() => {
    if (mockLocalStorage) {
      mockLocalStorage.clear();
    }
  });

  describe('Storage Layer', () => {
    test('init() sets localStorageAvailable to true', () => {
      storage.init();
      assert.strictEqual(storage.localStorageAvailable, true);
    });

    test('load() returns empty array when no data', () => {
      storage.init();
      const tasks = storage.load();
      assert.deepStrictEqual(tasks, []);
    });

    test('load() returns saved tasks', () => {
      storage.init();
      const testTasks = [
        {
          id: '123',
          text: 'Test task',
          completed: false,
          createdAt: Date.now()
        }
      ];
      mockLocalStorage.setItem(storage.STORAGE_KEY, JSON.stringify(testTasks));

      const loaded = storage.load();
      assert.deepStrictEqual(loaded, testTasks);
    });

    test('save() stores tasks in localStorage', () => {
      storage.init();
      const tasks = [
        {
          id: '456',
          text: 'Another task',
          completed: true,
          createdAt: Date.now()
        }
      ];

      storage.save(tasks);

      const stored = JSON.parse(mockLocalStorage.getItem(storage.STORAGE_KEY));
      assert.deepStrictEqual(stored, tasks);
    });

    test('load() handles corrupted data gracefully (returns [])', () => {
      storage.init();
      mockLocalStorage.setItem(storage.STORAGE_KEY, 'invalid json{');

      const tasks = storage.load();
      assert.deepStrictEqual(tasks, []);
    });

    test('load() handles non-array data (returns [])', () => {
      storage.init();
      mockLocalStorage.setItem(storage.STORAGE_KEY, JSON.stringify({ not: 'an array' }));

      const tasks = storage.load();
      assert.deepStrictEqual(tasks, []);
    });

    test('load() filters out tasks with missing fields', () => {
      storage.init();
      const invalidTasks = [
        { id: '1', text: 'Valid', completed: false, createdAt: 123 },
        { id: '2', text: 'Missing completed' }, // missing completed and createdAt
        { id: '3', completed: false, createdAt: 456 }, // missing text
        { text: 'Missing id', completed: true, createdAt: 789 } // missing id
      ];

      mockLocalStorage.setItem(storage.STORAGE_KEY, JSON.stringify(invalidTasks));

      const tasks = storage.load();
      assert.strictEqual(tasks.length, 1);
      assert.strictEqual(tasks[0].id, '1');
    });
  });

  describe('TodoStore', () => {
    test('addTask creates task with correct shape', () => {
      const store = new TodoStore();
      const beforeTime = Date.now();

      store.addTask('New task');

      const tasks = store.getTasks();
      assert.strictEqual(tasks.length, 1);

      const task = tasks[0];
      assert.strictEqual(typeof task.id, 'string');
      assert.strictEqual(task.text, 'New task');
      assert.strictEqual(task.completed, false);
      assert.strictEqual(typeof task.createdAt, 'number');
      assert.ok(task.createdAt >= beforeTime);
    });

    test('addTask trims whitespace', () => {
      const store = new TodoStore();

      store.addTask('  Trimmed task  ');

      const tasks = store.getTasks();
      assert.strictEqual(tasks[0].text, 'Trimmed task');
    });

    test('addTask rejects empty/whitespace text', () => {
      const store = new TodoStore();

      store.addTask('');
      assert.strictEqual(store.getTasks().length, 0);

      store.addTask('   ');
      assert.strictEqual(store.getTasks().length, 0);

      store.addTask('\t\n');
      assert.strictEqual(store.getTasks().length, 0);
    });

    test('toggleTask changes completed state', () => {
      const store = new TodoStore();
      store.addTask('Task to toggle');

      const task = store.getTasks()[0];
      assert.strictEqual(task.completed, false);

      store.toggleTask(task.id);
      assert.strictEqual(store.getTasks()[0].completed, true);

      store.toggleTask(task.id);
      assert.strictEqual(store.getTasks()[0].completed, false);
    });

    test('editTask updates task text', () => {
      const store = new TodoStore();
      store.addTask('Original text');

      const taskId = store.getTasks()[0].id;
      const result = store.editTask(taskId, 'Updated text');

      assert.strictEqual(result, true);
      assert.strictEqual(store.getTasks()[0].text, 'Updated text');
    });

    test('editTask trims whitespace', () => {
      const store = new TodoStore();
      store.addTask('Original');

      const taskId = store.getTasks()[0].id;
      store.editTask(taskId, '  Trimmed  ');

      assert.strictEqual(store.getTasks()[0].text, 'Trimmed');
    });

    test('editTask rejects empty string (returns false)', () => {
      const store = new TodoStore();
      store.addTask('Original text');

      const taskId = store.getTasks()[0].id;
      const result = store.editTask(taskId, '');

      assert.strictEqual(result, false);
      assert.strictEqual(store.getTasks()[0].text, 'Original text');
    });

    test('editTask rejects whitespace only (returns false)', () => {
      const store = new TodoStore();
      store.addTask('Original text');

      const taskId = store.getTasks()[0].id;
      const result = store.editTask(taskId, '   ');

      assert.strictEqual(result, false);
      assert.strictEqual(store.getTasks()[0].text, 'Original text');
    });

    test('deleteTask removes task', () => {
      const store = new TodoStore();
      store.addTask('Task 1');
      store.addTask('Task 2');

      assert.strictEqual(store.getTasks().length, 2);

      const taskId = store.getTasks()[0].id;
      store.deleteTask(taskId);

      assert.strictEqual(store.getTasks().length, 1);
      assert.strictEqual(store.getTasks()[0].text, 'Task 2');
    });

    test('clearCompleted only removes completed tasks', () => {
      const store = new TodoStore();
      store.addTask('Active 1');
      store.addTask('Completed 1');
      store.addTask('Active 2');
      store.addTask('Completed 2');

      const tasks = store.getTasks();
      store.toggleTask(tasks[1].id); // Mark as completed
      store.toggleTask(tasks[3].id); // Mark as completed

      assert.strictEqual(store.getTasks().length, 4);

      store.clearCompleted();

      const remaining = store.getTasks();
      assert.strictEqual(remaining.length, 2);
      assert.strictEqual(remaining[0].text, 'Active 1');
      assert.strictEqual(remaining[1].text, 'Active 2');
    });

    test('getActiveCount returns correct count', () => {
      const store = new TodoStore();

      assert.strictEqual(store.getActiveCount(), 0);

      store.addTask('Active 1');
      store.addTask('Active 2');
      store.addTask('To be completed');

      assert.strictEqual(store.getActiveCount(), 3);

      const tasks = store.getTasks();
      store.toggleTask(tasks[2].id);

      assert.strictEqual(store.getActiveCount(), 2);
    });

    test('getCompletedCount returns correct count', () => {
      const store = new TodoStore();

      assert.strictEqual(store.getCompletedCount(), 0);

      store.addTask('Task 1');
      store.addTask('Task 2');
      store.addTask('Task 3');

      assert.strictEqual(store.getCompletedCount(), 0);

      const tasks = store.getTasks();
      store.toggleTask(tasks[0].id);
      store.toggleTask(tasks[2].id);

      assert.strictEqual(store.getCompletedCount(), 2);
    });

    test('subscribe/notify pattern works', () => {
      const store = new TodoStore();
      let callCount = 0;
      let lastTasks = null;

      store.subscribe((tasks) => {
        callCount++;
        lastTasks = tasks;
      });

      store.addTask('New task');

      assert.ok(callCount > 0, 'Subscriber should be called');
      assert.strictEqual(lastTasks.length, 1);
      assert.strictEqual(lastTasks[0].text, 'New task');
    });

    test('unsubscribe removes subscriber', () => {
      const store = new TodoStore();
      let callCount = 0;

      const unsubscribe = store.subscribe(() => {
        callCount++;
      });

      store.addTask('Task 1');
      const countAfterFirst = callCount;

      unsubscribe();

      store.addTask('Task 2');

      assert.strictEqual(callCount, countAfterFirst, 'Unsubscribed callback should not be called');
    });
  });
});

// ==========================================
// EDGE CASE TESTS
// ==========================================

describe('Edge Case Tests', () => {
  let storage, TodoStore;
  let mockLocalStorage;

  beforeEach(() => {
    mockLocalStorage = {
      data: {},
      getItem(key) {
        return this.data[key] || null;
      },
      setItem(key, value) {
        this.data[key] = value;
      },
      removeItem(key) {
        delete this.data[key];
      },
      clear() {
        this.data = {};
      }
    };

    const mockGlobal = {
      localStorage: mockLocalStorage,
      console: {
        warn: () => {},
        error: () => {},
        log: () => {}
      }
    };

    let jsCode = parsed.script;

    // Remove the initialization code at the end (last 3 lines)
    // This prevents TodoUI from trying to access document
    jsCode = jsCode.replace(/\/\/ Initialize app[\s\S]*$/, '');

    const wrappedCode = `
      const localStorage = mockGlobal.localStorage;
      const console = mockGlobal.console;
      const document = mockGlobal.document || {};
      ${jsCode}
      return { storage, TodoStore };
    `;

    const fn = new Function('mockGlobal', wrappedCode);
    const result = fn(mockGlobal);

    storage = result.storage;
    TodoStore = result.TodoStore;
  });

  test('storage handles null data', () => {
    storage.init();
    mockLocalStorage.setItem(storage.STORAGE_KEY, null);

    const tasks = storage.load();
    assert.deepStrictEqual(tasks, []);
  });

  test('storage handles undefined data', () => {
    storage.init();
    // Don't set anything in localStorage

    const tasks = storage.load();
    assert.deepStrictEqual(tasks, []);
  });

  test('storage handles empty string', () => {
    storage.init();
    mockLocalStorage.setItem(storage.STORAGE_KEY, '');

    const tasks = storage.load();
    assert.deepStrictEqual(tasks, []);
  });

  test('addTask with various whitespace types is rejected', () => {
    const store = new TodoStore();

    store.addTask(' \t\n\r ');
    assert.strictEqual(store.getTasks().length, 0);
  });

  test('editTask on non-existent task returns false', () => {
    const store = new TodoStore();

    const result = store.editTask('non-existent-id', 'New text');
    assert.strictEqual(result, false);
  });

  test('toggleTask on non-existent task does nothing', () => {
    const store = new TodoStore();
    store.addTask('Task 1');

    const beforeLength = store.getTasks().length;
    store.toggleTask('non-existent-id');

    assert.strictEqual(store.getTasks().length, beforeLength);
  });

  test('deleteTask on non-existent task does nothing', () => {
    const store = new TodoStore();
    store.addTask('Task 1');
    store.addTask('Task 2');

    const beforeLength = store.getTasks().length;
    store.deleteTask('non-existent-id');

    assert.strictEqual(store.getTasks().length, beforeLength);
  });

  test('clearCompleted with no completed tasks does nothing', () => {
    const store = new TodoStore();
    store.addTask('Active 1');
    store.addTask('Active 2');

    assert.strictEqual(store.getTasks().length, 2);

    store.clearCompleted();

    assert.strictEqual(store.getTasks().length, 2);
  });

  test('clearCompleted with all completed tasks removes all', () => {
    const store = new TodoStore();
    store.addTask('Task 1');
    store.addTask('Task 2');

    const tasks = store.getTasks();
    store.toggleTask(tasks[0].id);
    store.toggleTask(tasks[1].id);

    store.clearCompleted();

    assert.strictEqual(store.getTasks().length, 0);
  });

  test('task IDs are unique', () => {
    const store = new TodoStore();

    for (let i = 0; i < 100; i++) {
      store.addTask(`Task ${i}`);
    }

    const tasks = store.getTasks();
    const ids = tasks.map(t => t.id);
    const uniqueIds = new Set(ids);

    assert.strictEqual(ids.length, uniqueIds.size, 'All task IDs should be unique');
  });

  test('tasks with invalid types are filtered on load', () => {
    storage.init();

    const invalidTasks = [
      { id: 123, text: 'Valid', completed: false, createdAt: 123 }, // id is number
      { id: '2', text: 456, completed: false, createdAt: 123 }, // text is number
      { id: '3', text: 'Valid', completed: 'yes', createdAt: 123 }, // completed is string
      { id: '4', text: 'Valid', completed: false, createdAt: '123' }, // createdAt is string
      { id: '5', text: 'Valid', completed: false, createdAt: 456 } // all valid
    ];

    mockLocalStorage.setItem(storage.STORAGE_KEY, JSON.stringify(invalidTasks));

    const tasks = storage.load();
    assert.strictEqual(tasks.length, 1);
    assert.strictEqual(tasks[0].id, '5');
  });

  test('multiple subscribers all get notified', () => {
    const store = new TodoStore();

    let count1 = 0;
    let count2 = 0;
    let count3 = 0;

    store.subscribe(() => count1++);
    store.subscribe(() => count2++);
    store.subscribe(() => count3++);

    store.addTask('New task');

    assert.ok(count1 > 0, 'First subscriber should be called');
    assert.ok(count2 > 0, 'Second subscriber should be called');
    assert.ok(count3 > 0, 'Third subscriber should be called');
  });
});
