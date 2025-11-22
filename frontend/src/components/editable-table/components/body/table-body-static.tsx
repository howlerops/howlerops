import { ColumnDef,Row } from '@tanstack/react-table';
import React from 'react';

import { TableRow } from '../../../../types/table';
import { MemoizedVirtualRow } from './virtual-row';

interface TableBodyStaticProps {
  rows: Row<TableRow>[];
  columns: ColumnDef<TableRow>[];
  onRowClick?: (rowId: string, rowData: TableRow) => void;
}

export const TableBodyStatic: React.FC<TableBodyStaticProps> = ({
  rows,
  columns,
  onRowClick,
}) => {
  return (
    <tbody>
      {rows.map(row => (
        <MemoizedVirtualRow
          key={row.id}
          row={row}
          columns={columns}
          state={undefined}
          actions={undefined}
          tableColumns={undefined}
          isVirtual={false}
          onRowClick={onRowClick}
        />
      ))}
    </tbody>
  );
};
