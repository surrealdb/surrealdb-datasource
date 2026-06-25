import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { DataSourceSettings } from '@grafana/data';
import { ConfigEditor } from './ConfigEditor';
import { SurrealDataSourceOptions, SurrealSecureJsonData } from '../types';

function makeOptions(
  jsonData: Partial<SurrealDataSourceOptions> = {}
): DataSourceSettings<SurrealDataSourceOptions, SurrealSecureJsonData> {
  return {
    jsonData,
    secureJsonData: {},
    secureJsonFields: {},
  } as DataSourceSettings<SurrealDataSourceOptions, SurrealSecureJsonData>;
}

describe('ConfigEditor', () => {
  it('does not show the obsolete SurrealDB v2.0 compatibility warning', () => {
    render(<ConfigEditor options={makeOptions()} onOptionsChange={jest.fn()} />);

    expect(screen.queryByText(/does not support SurrealDB v2\.0/i)).not.toBeInTheDocument();
  });

  it('renders the Access field and no longer the Scope field', () => {
    render(<ConfigEditor options={makeOptions()} onOptionsChange={jest.fn()} />);

    expect(screen.getByPlaceholderText('Access')).toBeInTheDocument();
    expect(screen.queryByPlaceholderText('Scope')).not.toBeInTheDocument();
  });

  it('writes the access method to jsonData', () => {
    const onOptionsChange = jest.fn();
    render(<ConfigEditor options={makeOptions()} onOptionsChange={onOptionsChange} />);

    fireEvent.change(screen.getByPlaceholderText('Access'), { target: { value: 'account' } });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({ jsonData: expect.objectContaining({ access: 'account' }) })
    );
  });

  it('writes the endpoint to jsonData', () => {
    const onOptionsChange = jest.fn();
    render(<ConfigEditor options={makeOptions()} onOptionsChange={onOptionsChange} />);

    fireEvent.change(screen.getByPlaceholderText('ws://localhost:8000/rpc'), {
      target: { value: 'ws://surreal:8000' },
    });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({ jsonData: expect.objectContaining({ endpoint: 'ws://surreal:8000' }) })
    );
  });

  it('writes the password to secureJsonData', () => {
    const onOptionsChange = jest.fn();
    render(<ConfigEditor options={makeOptions()} onOptionsChange={onOptionsChange} />);

    fireEvent.change(screen.getByPlaceholderText('Password'), { target: { value: 'secret' } });

    expect(onOptionsChange).toHaveBeenCalledWith(
      expect.objectContaining({ secureJsonData: expect.objectContaining({ password: 'secret' }) })
    );
  });
});
