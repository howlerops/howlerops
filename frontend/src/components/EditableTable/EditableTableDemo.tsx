import React, { useState, useCallback, useMemo } from 'react';
import { EditableTable } from './EditableTable';
import { TableRow, TableColumn, ExportOptions } from '../../types/table';

// Sample data
const generateSampleData = (count: number): TableRow[] => {
  const data: TableRow[] = [];
  const statuses = ['Active', 'Inactive', 'Pending', 'Suspended'];
  const departments = ['Engineering', 'Sales', 'Marketing', 'HR', 'Finance'];

  for (let i = 0; i < count; i++) {
    data.push({
      id: `row-${i + 1}`,
      name: `User ${i + 1}`,
      email: `user${i + 1}@example.com`,
      age: 20 + Math.floor(Math.random() * 40),
      salary: 40000 + Math.floor(Math.random() * 100000),
      department: departments[Math.floor(Math.random() * departments.length)],
      status: statuses[Math.floor(Math.random() * statuses.length)],
      isActive: Math.random() > 0.5,
      joinDate: new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
      notes: i % 5 === 0 ? `Long note for user ${i + 1}. This is a longer text field that might contain multiple lines of information about the user.` : `Note ${i + 1}`,
    });
  }

  return data;
};

// Column definitions
const columns: TableColumn[] = [
  {
    id: 'name',
    accessorKey: 'name',
    header: 'Name',
    type: 'text',
    sortable: true,
    filterable: true,
    editable: true,
    required: true,
    minWidth: 120,
    validation: {
      min: 2,
      max: 50,
      message: 'Name must be between 2 and 50 characters',
    },
  },
  {
    id: 'email',
    accessorKey: 'email',
    header: 'Email',
    type: 'text',
    sortable: true,
    filterable: true,
    editable: true,
    required: true,
    minWidth: 200,
    validation: {
      pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
      message: 'Please enter a valid email address',
    },
  },
  {
    id: 'age',
    accessorKey: 'age',
    header: 'Age',
    type: 'number',
    sortable: true,
    filterable: true,
    editable: true,
    width: 80,
    validation: {
      min: 18,
      max: 120,
      message: 'Age must be between 18 and 120',
    },
  },
  {
    id: 'salary',
    accessorKey: 'salary',
    header: 'Salary',
    type: 'number',
    sortable: true,
    filterable: true,
    editable: true,
    minWidth: 120,
    validation: {
      min: 20000,
      max: 500000,
      message: 'Salary must be between $20,000 and $500,000',
    },
  },
  {
    id: 'department',
    accessorKey: 'department',
    header: 'Department',
    type: 'select',
    sortable: true,
    filterable: true,
    editable: true,
    width: 130,
    options: ['Engineering', 'Sales', 'Marketing', 'HR', 'Finance'],
  },
  {
    id: 'status',
    accessorKey: 'status',
    header: 'Status',
    type: 'select',
    sortable: true,
    filterable: true,
    editable: true,
    width: 100,
    options: ['Active', 'Inactive', 'Pending', 'Suspended'],
  },
  {
    id: 'isActive',
    accessorKey: 'isActive',
    header: 'Active',
    type: 'boolean',
    sortable: true,
    filterable: true,
    editable: true,
    width: 80,
  },
  {
    id: 'joinDate',
    accessorKey: 'joinDate',
    header: 'Join Date',
    type: 'date',
    sortable: true,
    filterable: true,
    editable: true,
    width: 120,
  },
  {
    id: 'notes',
    accessorKey: 'notes',
    header: 'Notes',
    type: 'text',
    sortable: false,
    filterable: true,
    editable: true,
    minWidth: 200,
    validation: {
      max: 500,
      message: 'Notes cannot exceed 500 characters',
    },
  },
];

export const EditableTableDemo: React.FC = () => {
  const [data, setData] = useState<TableRow[]>(() => generateSampleData(1000));
  const [loading, setLoading] = useState(false); // eslint-disable-line @typescript-eslint/no-unused-vars
  const [selectedRowIds, setSelectedRowIds] = useState<string[]>([]);

  // Simulate API call for cell editing
  const handleCellEdit = useCallback(async (
    rowId: string,
    columnId: string,
    value: unknown
  ): Promise<boolean> => {
    // Simulate network delay
    await new Promise(resolve => setTimeout(resolve, 100 + Math.random() * 400));

    // Simulate occasional failures (10% chance)
    if (Math.random() < 0.1) {
      throw new Error('Simulated network error');
    }

    console.log(`Updating ${rowId}.${columnId} to:`, value);
    return true;
  }, []);

  const handleDataChange = useCallback((newData: TableRow[]) => {
    console.log('Data changed:', newData.length, 'rows');
  }, []);

  const handleRowSelect = useCallback((selectedIds: string[]) => {
    setSelectedRowIds(selectedIds);
    console.log('Selected rows:', selectedIds);
  }, []);

  const handleExport = useCallback((options: ExportOptions) => { // eslint-disable-line @typescript-eslint/no-unused-vars
    console.log('Exporting with options:', options);

    const exportData = options.selectedOnly && selectedRowIds.length > 0
      ? data.filter(row => selectedRowIds.includes(row.id))
      : data;

    // In a real application, you would use the exportData utility
    // exportData(exportData, columns, options);

    alert(`Would export ${exportData.length} rows as ${options.format.toUpperCase()}`);
  }, [data, selectedRowIds]);


  const handleAddRandomRow = useCallback(() => {
    const newRow: TableRow = {
      id: `row-${Date.now()}`,
      name: `New User ${Date.now()}`,
      email: `newuser${Date.now()}@example.com`,
      age: 25,
      salary: 60000,
      department: 'Engineering',
      status: 'Active',
      isActive: true,
      joinDate: new Date().toISOString().split('T')[0],
      notes: 'Newly added user',
    };

    setData(prev => [newRow, ...prev]);
  }, []);

  const handleDeleteSelected = useCallback(() => {
    if (selectedRowIds.length === 0) {
      alert('No rows selected');
      return;
    }

    const confirmDelete = window.confirm(
      `Are you sure you want to delete ${selectedRowIds.length} row(s)?`
    );

    if (confirmDelete) {
      setData(prev => prev.filter(row => !selectedRowIds.includes(row.id)));
      setSelectedRowIds([]);
    }
  }, [selectedRowIds]);

  const customToolbarActions = useMemo(() => (
    <div className="flex items-center gap-2">
      <button
        onClick={handleAddRandomRow}
        className="px-3 py-2 bg-green-600 text-white rounded hover:bg-green-700 transition-colors text-sm"
      >
        Add Row
      </button>
      {selectedRowIds.length > 0 && (
        <button
          onClick={handleDeleteSelected}
          className="px-3 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors text-sm"
        >
          Delete Selected ({selectedRowIds.length})
        </button>
      )}
    </div>
  ), [handleAddRandomRow, handleDeleteSelected, selectedRowIds.length]);

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">EditableTable Demo</h1>
        <p className="text-gray-600 mt-2">
          A high-performance editable table component with virtual scrolling,
          inline editing, keyboard navigation, and more.
        </p>
      </div>

      {/* Table stats */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="text-2xl font-bold text-blue-600">{data.length.toLocaleString()}</div>
          <div className="text-sm text-gray-600">Total Rows</div>
        </div>
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="text-2xl font-bold text-green-600">{selectedRowIds.length}</div>
          <div className="text-sm text-gray-600">Selected</div>
        </div>
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="text-2xl font-bold text-purple-600">{columns.length}</div>
          <div className="text-sm text-gray-600">Columns</div>
        </div>
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="text-2xl font-bold text-orange-600">
            {columns.filter(col => col.editable).length}
          </div>
          <div className="text-sm text-gray-600">Editable</div>
        </div>
      </div>

      {/* Instructions */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="font-semibold text-blue-900 mb-2">How to use:</h3>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>• <strong>Double-click</strong> any cell to edit (except read-only columns)</li>
          <li>• Use <strong>Tab</strong> or <strong>Enter</strong> to navigate between cells</li>
          <li>• <strong>Escape</strong> to cancel editing</li>
          <li>• <strong>Ctrl+C</strong> to copy, <strong>Ctrl+V</strong> to paste</li>
          <li>• <strong>Ctrl+Z</strong> to undo, <strong>Ctrl+Y</strong> to redo</li>
          <li>• Click column headers to sort, use filter icons to filter</li>
          <li>• Drag column borders to resize</li>
          <li>• Use checkboxes for multi-select operations</li>
        </ul>
      </div>

      {/* Main Table */}
      <div className="bg-white border border-gray-200 rounded-lg overflow-hidden">
        <EditableTable
          data={data}
          columns={columns}
          onDataChange={handleDataChange}
          onCellEdit={handleCellEdit}
          onRowSelect={handleRowSelect}
          loading={loading}
          virtualScrolling={true}
          estimateSize={45}
          height={600}
          enableMultiSelect={true}
          enableColumnResizing={true}
          enableGlobalFilter={true}
          enableExport={true}
          toolbar={
            <div className="flex items-center justify-between p-4 bg-white border-b border-gray-200">
              <div className="flex items-center gap-4 flex-1">
                {/* Search will be handled by the table internally */}
                <div className="text-sm text-gray-600">
                  Use the search box to filter data across all columns
                </div>
              </div>
              {customToolbarActions}
            </div>
          }
        />
      </div>

      {/* Performance tips */}
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-4">
        <h3 className="font-semibold text-gray-900 mb-2">Performance Features:</h3>
        <ul className="text-sm text-gray-700 space-y-1">
          <li>• <strong>Virtual scrolling</strong> handles 100,000+ rows smoothly</li>
          <li>• <strong>React.memo</strong> prevents unnecessary re-renders</li>
          <li>• <strong>Debounced search</strong> reduces API calls</li>
          <li>• <strong>Optimistic updates</strong> with automatic rollback on failure</li>
          <li>• <strong>Request animation frame</strong> for smooth scrolling</li>
          <li>• <strong>Keyboard navigation</strong> with efficient event handling</li>
        </ul>
      </div>
    </div>
  );
};