/**
 * AGGridTable Usage Examples
 *
 * Demonstrates various use cases and features of the AG Grid-based table component.
 */

import React, { useState } from 'react';

import { TableColumn, TableRow } from '../../types/table';
import { AGGridTable } from './ag-grid-table';

/**
 * Example 1: Basic Table
 */
export function BasicTableExample() {
  const columns: TableColumn[] = [
    {
      id: 'id',
      header: 'ID',
      type: 'number',
      editable: false,
      isPrimaryKey: true,
      width: 80,
    },
    {
      id: 'name',
      header: 'Name',
      type: 'text',
      editable: true,
      required: true,
    },
    {
      id: 'email',
      header: 'Email',
      type: 'text',
      editable: true,
      validation: {
        pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
        message: 'Invalid email address',
      },
    },
    {
      id: 'active',
      header: 'Active',
      type: 'boolean',
      editable: true,
      width: 100,
    },
  ];

  const [data, setData] = useState<TableRow[]>([
    { __rowId: '1', id: 1, name: 'John Doe', email: 'john@example.com', active: true },
    { __rowId: '2', id: 2, name: 'Jane Smith', email: 'jane@example.com', active: false },
    { __rowId: '3', id: 3, name: 'Bob Johnson', email: 'bob@example.com', active: true },
  ]);

  return (
    <AGGridTable
      data={data}
      columns={columns}
      height={400}
      onDataChange={setData}
      onCellEdit={async (rowId, columnId, value) => {
        console.log('Edit:', { rowId, columnId, value });
        // Simulate API call
        await new Promise(resolve => setTimeout(resolve, 500));
        return true;
      }}
    />
  );
}

/**
 * Example 2: Table with Selection
 */
export function TableWithSelectionExample() {
  const [selectedRows, setSelectedRows] = useState<string[]>([]);

  const columns: TableColumn[] = [
    { id: 'id', header: 'ID', type: 'number', width: 80 },
    { id: 'product', header: 'Product', type: 'text' },
    { id: 'price', header: 'Price', type: 'number' },
    { id: 'inStock', header: 'In Stock', type: 'boolean' },
  ];

  const data: TableRow[] = [
    { __rowId: '1', id: 1, product: 'Laptop', price: 999.99, inStock: true },
    { __rowId: '2', id: 2, product: 'Mouse', price: 29.99, inStock: true },
    { __rowId: '3', id: 3, product: 'Keyboard', price: 79.99, inStock: false },
  ];

  return (
    <div>
      <div className="mb-4">
        <p>Selected: {selectedRows.length} items</p>
        {selectedRows.length > 0 && (
          <button
            onClick={() => console.log('Delete:', selectedRows)}
            className="mt-2 px-4 py-2 bg-destructive text-destructive-foreground rounded"
          >
            Delete Selected
          </button>
        )}
      </div>

      <AGGridTable
        data={data}
        columns={columns}
        height={300}
        enableMultiSelect={true}
        onRowSelect={setSelectedRows}
      />
    </div>
  );
}

/**
 * Example 3: Table with Custom Renderers
 */
export function TableWithCustomRenderersExample() {
  const columns: TableColumn[] = [
    { id: 'id', header: 'ID', type: 'number', width: 80 },
    { id: 'name', header: 'User', type: 'text' },
    { id: 'status', header: 'Status', type: 'select', options: ['active', 'pending', 'inactive'] },
    { id: 'score', header: 'Score', type: 'number' },
  ];

  const data: TableRow[] = [
    { __rowId: '1', id: 1, name: 'Alice', status: 'active', score: 95 },
    { __rowId: '2', id: 2, name: 'Bob', status: 'pending', score: 72 },
    { __rowId: '3', id: 3, name: 'Charlie', status: 'inactive', score: 88 },
  ];

  return (
    <AGGridTable
      data={data}
      columns={columns}
      height={300}
      customCellRenderers={{
        status: (value) => {
          const colors = {
            active: 'bg-green-500/20 text-green-500',
            pending: 'bg-yellow-500/20 text-yellow-500',
            inactive: 'bg-red-500/20 text-red-500',
          };

          const colorClass = colors[value as keyof typeof colors] || '';

          return (
            <div className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${colorClass}`}>
              {value}
            </div>
          );
        },
        score: (value) => {
          const score = Number(value);
          const color = score >= 90 ? 'text-green-500' : score >= 70 ? 'text-yellow-500' : 'text-red-500';

          return (
            <div className={`font-semibold ${color}`}>
              {score}%
            </div>
          );
        },
      }}
    />
  );
}

/**
 * Example 4: Table with Toolbar and Footer
 */
export function TableWithToolbarExample() {
  const [searchQuery, setSearchQuery] = useState('');

  const columns: TableColumn[] = [
    { id: 'id', header: 'ID', type: 'number', width: 80 },
    { id: 'title', header: 'Title', type: 'text', editable: true },
    { id: 'author', header: 'Author', type: 'text', editable: true },
    { id: 'published', header: 'Published', type: 'boolean' },
  ];

  const data: TableRow[] = [
    { __rowId: '1', id: 1, title: 'React Guide', author: 'John', published: true },
    { __rowId: '2', id: 2, title: 'TypeScript Handbook', author: 'Jane', published: false },
    { __rowId: '3', id: 3, title: 'CSS Tricks', author: 'Bob', published: true },
  ];

  return (
    <AGGridTable
      data={data}
      columns={columns}
      height={400}
      toolbar={(context) => (
        <div className="flex items-center justify-between p-3 border-b border-border">
          <div className="flex items-center gap-3">
            <input
              type="text"
              placeholder="Search..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                context.actions.updateGlobalFilter(e.target.value);
              }}
              className="px-3 py-1 border border-border rounded bg-background text-foreground"
            />

            <button
              onClick={() => context.actions.selectAllRows(true)}
              className="px-3 py-1 text-sm border border-border rounded hover:bg-accent"
            >
              Select All
            </button>

            <button
              onClick={() => context.actions.resetTable()}
              className="px-3 py-1 text-sm border border-border rounded hover:bg-accent"
            >
              Reset
            </button>
          </div>

          <div className="text-sm text-muted-foreground">
            {context.state.hasSelection && (
              <span>{context.state.selectedRows.length} selected</span>
            )}
          </div>
        </div>
      )}
      footer={(context) => (
        <div className="flex items-center justify-between p-3 border-t border-border text-sm">
          <div>
            Total: {context.data.length} rows
          </div>

          {context.state.hasDirtyRows && (
            <div className="flex items-center gap-2 text-warning">
              <span>{context.state.dirtyRows.size} unsaved changes</span>
              <button
                onClick={() => context.actions.clearDirtyRows()}
                className="px-2 py-1 text-xs border border-warning rounded hover:bg-warning/10"
              >
                Discard
              </button>
            </div>
          )}
        </div>
      )}
    />
  );
}

/**
 * Example 5: Advanced Configuration
 */
export function AdvancedTableExample() {
  const columns: TableColumn[] = [
    {
      id: 'id',
      header: 'ID',
      type: 'number',
      sticky: 'left',
      isPrimaryKey: true,
      width: 80,
      editable: false,
    },
    {
      id: 'code',
      header: 'Code',
      type: 'text',
      monospace: true,
      clipContent: true,
      width: 150,
    },
    {
      id: 'description',
      header: 'Description',
      type: 'text',
      longText: true,
      wrapContent: true,
      minWidth: 200,
    },
    {
      id: 'amount',
      header: 'Amount',
      type: 'number',
      validation: {
        min: 0,
        max: 10000,
        message: 'Amount must be between 0 and 10,000',
      },
      width: 120,
    },
    {
      id: 'category',
      header: 'Category',
      type: 'select',
      options: ['Electronics', 'Furniture', 'Clothing', 'Food'],
      width: 150,
    },
    {
      id: 'active',
      header: 'Active',
      type: 'boolean',
      sticky: 'right',
      width: 100,
    },
  ];

  const data: TableRow[] = Array.from({ length: 100 }, (_, i) => ({
    __rowId: String(i + 1),
    id: i + 1,
    code: `ITEM-${String(i + 1).padStart(4, '0')}`,
    description: `This is a sample description for item ${i + 1}. It can be quite long and will wrap if needed.`,
    amount: Math.floor(Math.random() * 10000),
    category: ['Electronics', 'Furniture', 'Clothing', 'Food'][i % 4],
    active: i % 3 !== 0,
  }));

  return (
    <AGGridTable
      data={data}
      columns={columns}
      height={600}
      enableMultiSelect={true}
      enableColumnResizing={true}
      enableGlobalFilter={true}
      onCellEdit={async (rowId, columnId, value) => {
        console.log('Edit:', { rowId, columnId, value });
        await new Promise(resolve => setTimeout(resolve, 300));
        return true;
      }}
      onRowClick={(rowId, rowData) => {
        console.log('Click:', rowId, rowData);
      }}
      onRowInspect={(rowId, rowData) => {
        console.log('Double-click (inspect):', rowId, rowData);
      }}
      onDirtyChange={(dirtyRowIds) => {
        console.log('Dirty rows:', dirtyRowIds);
      }}
    />
  );
}

/**
 * Example 6: Large Dataset (Performance Test)
 */
export function LargeDatasetExample() {
  const columns: TableColumn[] = [
    { id: 'id', header: 'ID', type: 'number', width: 80 },
    { id: 'firstName', header: 'First Name', type: 'text' },
    { id: 'lastName', header: 'Last Name', type: 'text' },
    { id: 'email', header: 'Email', type: 'text', width: 250 },
    { id: 'age', header: 'Age', type: 'number', width: 80 },
    { id: 'department', header: 'Department', type: 'text' },
    { id: 'salary', header: 'Salary', type: 'number', width: 120 },
    { id: 'active', header: 'Active', type: 'boolean', width: 100 },
  ];

  // Generate 10,000 rows for performance testing
  const data: TableRow[] = Array.from({ length: 10000 }, (_, i) => ({
    __rowId: String(i + 1),
    id: i + 1,
    firstName: `First${i}`,
    lastName: `Last${i}`,
    email: `user${i}@example.com`,
    age: 20 + (i % 50),
    department: ['Engineering', 'Sales', 'Marketing', 'HR'][i % 4],
    salary: 50000 + (i * 100),
    active: i % 5 !== 0,
  }));

  return (
    <div>
      <div className="mb-4 p-4 bg-muted rounded">
        <h3 className="font-semibold mb-2">Performance Test</h3>
        <p className="text-sm text-muted-foreground">
          This table renders 10,000 rows using AG Grid's virtualization.
          Scroll quickly to test performance - no white chunks should appear.
        </p>
      </div>

      <AGGridTable
        data={data}
        columns={columns}
        height={600}
        enableMultiSelect={true}
        enableColumnResizing={true}
      />
    </div>
  );
}
