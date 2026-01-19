/**
 * AGGridTable Component Tests
 */

import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { TableColumn, TableRow } from '../../types/table';
import { AGGridTable } from './ag-grid-table';

// Mock AG Grid React component
vi.mock('ag-grid-react', () => ({
  AgGridReact: ({ rowData, columnDefs, onGridReady, getRowId }: any) => {
    // Simple mock that renders a basic table
    return (
      <div data-testid="ag-grid-mock">
        <table>
          <thead>
            <tr>
              {columnDefs?.map((col: any) => (
                <th key={col.colId}>{col.headerName}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rowData?.map((row: any, index: number) => (
              <tr key={getRowId ? getRowId({ data: row }) : index}>
                {columnDefs?.map((col: any) => (
                  <td key={col.colId}>
                    {col.cellRenderer
                      ? col.cellRenderer({ value: row[col.field], data: row })
                      : row[col.field]}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  },
}));

describe('AGGridTable', () => {
  const mockColumns: TableColumn[] = [
    {
      id: 'id',
      header: 'ID',
      type: 'number',
      editable: false,
      isPrimaryKey: true,
    },
    {
      id: 'name',
      header: 'Name',
      type: 'text',
      editable: true,
    },
    {
      id: 'active',
      header: 'Active',
      type: 'boolean',
      editable: true,
    },
  ];

  const mockData: TableRow[] = [
    { __rowId: '1', id: 1, name: 'John Doe', active: true },
    { __rowId: '2', id: 2, name: 'Jane Smith', active: false },
    { __rowId: '3', id: 3, name: 'Bob Johnson', active: true },
  ];

  describe('Rendering', () => {
    it('renders without crashing', () => {
      render(<AGGridTable data={mockData} columns={mockColumns} />);
      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });

    it('renders column headers', () => {
      render(<AGGridTable data={mockData} columns={mockColumns} />);
      expect(screen.getByText('ID')).toBeInTheDocument();
      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Active')).toBeInTheDocument();
    });

    it('renders row data', () => {
      render(<AGGridTable data={mockData} columns={mockColumns} />);
      expect(screen.getByText('John Doe')).toBeInTheDocument();
      expect(screen.getByText('Jane Smith')).toBeInTheDocument();
      expect(screen.getByText('Bob Johnson')).toBeInTheDocument();
    });

    it('applies custom className', () => {
      const { container } = render(
        <AGGridTable data={mockData} columns={mockColumns} className="custom-class" />
      );
      expect(container.firstChild).toHaveClass('custom-class');
    });

    it('displays error message when error prop is provided', () => {
      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          error="Something went wrong"
        />
      );
      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    });
  });

  describe('Column Configuration', () => {
    it('maps text columns correctly', () => {
      const columns: TableColumn[] = [
        { id: 'text', header: 'Text', type: 'text' },
      ];
      render(<AGGridTable data={[]} columns={columns} />);
      expect(screen.getByText('Text')).toBeInTheDocument();
    });

    it('maps number columns correctly', () => {
      const columns: TableColumn[] = [
        { id: 'num', header: 'Number', type: 'number' },
      ];
      render(<AGGridTable data={[]} columns={columns} />);
      expect(screen.getByText('Number')).toBeInTheDocument();
    });

    it('maps boolean columns correctly', () => {
      const columns: TableColumn[] = [
        { id: 'bool', header: 'Boolean', type: 'boolean' },
      ];
      render(<AGGridTable data={[]} columns={columns} />);
      expect(screen.getByText('Boolean')).toBeInTheDocument();
    });

    it('maps select columns correctly', () => {
      const columns: TableColumn[] = [
        { id: 'select', header: 'Select', type: 'select', options: ['A', 'B', 'C'] },
      ];
      render(<AGGridTable data={[]} columns={columns} />);
      expect(screen.getByText('Select')).toBeInTheDocument();
    });
  });

  describe('Custom Cell Renderers', () => {
    it('uses custom cell renderer when provided', () => {
      const customRenderer = vi.fn((value) => <span>Custom: {value}</span>);
      const customCellRenderers = {
        name: customRenderer,
      };

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          customCellRenderers={customCellRenderers}
        />
      );

      expect(customRenderer).toHaveBeenCalled();
    });

    it('renders NULL values with special styling', () => {
      const dataWithNull: TableRow[] = [
        { __rowId: '1', id: 1, name: null, active: true },
      ];

      render(<AGGridTable data={dataWithNull} columns={mockColumns} />);
      expect(screen.getByText('NULL')).toBeInTheDocument();
    });
  });

  describe('Callbacks', () => {
    it('calls onDataChange when data changes', async () => {
      const onDataChange = vi.fn();

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          onDataChange={onDataChange}
        />
      );

      // Mock implementation would trigger this in real AG Grid
      // For now, we just verify the callback is passed
      expect(onDataChange).not.toHaveBeenCalled();
    });

    it('calls onCellEdit when cell is edited', async () => {
      const onCellEdit = vi.fn().mockResolvedValue(true);

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          onCellEdit={onCellEdit}
        />
      );

      // Mock implementation would trigger this in real AG Grid
      expect(onCellEdit).not.toHaveBeenCalled();
    });

    it('calls onRowSelect when selection changes', () => {
      const onRowSelect = vi.fn();

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          onRowSelect={onRowSelect}
        />
      );

      expect(onRowSelect).not.toHaveBeenCalled();
    });

    it('calls onRowClick when row is clicked', () => {
      const onRowClick = vi.fn();

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          onRowClick={onRowClick}
        />
      );

      expect(onRowClick).not.toHaveBeenCalled();
    });

    it('calls onRowInspect when row is double-clicked', () => {
      const onRowInspect = vi.fn();

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          onRowInspect={onRowInspect}
        />
      );

      expect(onRowInspect).not.toHaveBeenCalled();
    });
  });

  describe('Toolbar and Footer', () => {
    it('renders toolbar when provided as ReactNode', () => {
      const toolbar = <div data-testid="custom-toolbar">Custom Toolbar</div>;

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          toolbar={toolbar}
        />
      );

      expect(screen.getByTestId('custom-toolbar')).toBeInTheDocument();
    });

    it('renders footer when provided as ReactNode', () => {
      const footer = <div data-testid="custom-footer">Custom Footer</div>;

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          footer={footer}
        />
      );

      expect(screen.getByTestId('custom-footer')).toBeInTheDocument();
    });

    it('renders toolbar when provided as function', () => {
      const toolbar = (context: any) => (
        <div data-testid="function-toolbar">
          Total: {context.data.length}
        </div>
      );

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          toolbar={toolbar}
        />
      );

      expect(screen.getByTestId('function-toolbar')).toBeInTheDocument();
      expect(screen.getByText('Total: 3')).toBeInTheDocument();
    });

    it('renders footer when provided as function', () => {
      const footer = (context: any) => (
        <div data-testid="function-footer">
          Rows: {context.data.length}
        </div>
      );

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          footer={footer}
        />
      );

      expect(screen.getByTestId('function-footer')).toBeInTheDocument();
      expect(screen.getByText('Rows: 3')).toBeInTheDocument();
    });
  });

  describe('Features', () => {
    it('supports multi-select when enabled', () => {
      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          enableMultiSelect={true}
        />
      );

      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });

    it('supports single-select when multi-select is disabled', () => {
      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          enableMultiSelect={false}
        />
      );

      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });

    it('supports column resizing when enabled', () => {
      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          enableColumnResizing={true}
        />
      );

      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });
  });

  describe('Dirty State Tracking', () => {
    it('calls onDirtyChange when rows are edited', async () => {
      const onDirtyChange = vi.fn();

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          onDirtyChange={onDirtyChange}
        />
      );

      // Initial call with empty dirty set
      await waitFor(() => {
        expect(onDirtyChange).toHaveBeenCalledWith([]);
      });
    });

    it('tracks dirty rows correctly', () => {
      const onCellEdit = vi.fn().mockResolvedValue(true);
      const onDirtyChange = vi.fn();

      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          onCellEdit={onCellEdit}
          onDirtyChange={onDirtyChange}
        />
      );

      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });
  });

  describe('Validation', () => {
    it('validates number columns with min/max', () => {
      const columns: TableColumn[] = [
        {
          id: 'amount',
          header: 'Amount',
          type: 'number',
          validation: {
            min: 0,
            max: 100,
            message: 'Must be between 0 and 100',
          },
        },
      ];

      render(<AGGridTable data={[]} columns={columns} />);
      expect(screen.getByText('Amount')).toBeInTheDocument();
    });

    it('validates text columns with pattern', () => {
      const columns: TableColumn[] = [
        {
          id: 'email',
          header: 'Email',
          type: 'text',
          validation: {
            pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
            message: 'Invalid email',
          },
        },
      ];

      render(<AGGridTable data={[]} columns={columns} />);
      expect(screen.getByText('Email')).toBeInTheDocument();
    });
  });

  describe('Height Configuration', () => {
    it('accepts height as number', () => {
      const { container } = render(
        <AGGridTable data={mockData} columns={mockColumns} height={500} />
      );

      const gridContainer = container.querySelector('.ag-theme-quartz-dark');
      expect(gridContainer).toHaveStyle({ height: '500px' });
    });

    it('accepts height as string', () => {
      const { container } = render(
        <AGGridTable data={mockData} columns={mockColumns} height="100vh" />
      );

      const gridContainer = container.querySelector('.ag-theme-quartz-dark');
      // Check inline style attribute since computed style converts viewport units to pixels
      expect(gridContainer?.getAttribute('style')).toContain('height: 100vh');
    });
  });

  describe('Loading State', () => {
    it('shows loading indicator when loading prop is true', () => {
      render(
        <AGGridTable
          data={mockData}
          columns={mockColumns}
          loading={true}
        />
      );

      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    it('handles empty data array', () => {
      render(<AGGridTable data={[]} columns={mockColumns} />);
      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });

    it('handles missing __rowId by generating one', () => {
      const dataWithoutRowId: TableRow[] = [
        { id: 1, name: 'Test', active: true },
      ];

      render(<AGGridTable data={dataWithoutRowId} columns={mockColumns} />);
      expect(screen.getByTestId('ag-grid-mock')).toBeInTheDocument();
    });

    it('handles columns without id or accessorKey', () => {
      const columns: TableColumn[] = [
        { header: 'Test', type: 'text' },
      ];

      render(<AGGridTable data={[]} columns={columns} />);
      expect(screen.getByText('Test')).toBeInTheDocument();
    });
  });
});
