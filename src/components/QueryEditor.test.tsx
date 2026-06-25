import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryEditor } from './QueryEditor';
import { SurrealQuery } from '../types';

// CodeEditor wraps Monaco, which does not run in jsdom; replace it with a plain
// textarea that forwards onBlur, which is all this component relies on.
jest.mock('@grafana/ui', () => ({
  CodeEditor: ({ value, onBlur }: { value: string; onBlur: (v: string) => void }) => (
    <textarea aria-label="query-editor" defaultValue={value} onBlur={(e) => onBlur(e.currentTarget.value)} />
  ),
}));

describe('QueryEditor', () => {
  it('emits the edited SurrealQL on blur, preserving other query fields', () => {
    const onChange = jest.fn();
    const query = { refId: 'A', rawSql: 'SELECT * FROM person' } as SurrealQuery;

    render(
      <QueryEditor
        query={query}
        onChange={onChange}
        onRunQuery={jest.fn()}
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        datasource={{} as any}
      />
    );

    fireEvent.blur(screen.getByLabelText('query-editor'), {
      target: { value: 'SELECT name FROM person' },
    });

    expect(onChange).toHaveBeenCalledWith({ refId: 'A', rawSql: 'SELECT name FROM person' });
  });
});
