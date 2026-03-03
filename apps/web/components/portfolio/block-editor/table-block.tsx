'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Loader2, Plus, Trash2, GripVertical, Columns, Rows } from 'lucide-react';
import { TableBlockPayload } from '@/lib/types';
import { cn } from '@/lib/utils';

interface TableBlockEditorProps {
  payload: TableBlockPayload;
  isEditing: boolean;
  onSave: (payload: TableBlockPayload) => void;
  onCancel: () => void;
  isSaving: boolean;
}

export function TableBlockEditor({ payload, isEditing, onSave, onCancel, isSaving }: TableBlockEditorProps) {
  // Initialize with default values if empty
  const initialHeaders = payload.headers?.length > 0 ? payload.headers : ['Kolom 1', 'Kolom 2'];
  const initialRows = payload.rows?.length > 0 ? payload.rows : [['', '']];

  const [headers, setHeaders] = useState<string[]>(initialHeaders);
  const [rows, setRows] = useState<string[][]>(initialRows);

  // View mode - display the table
  if (!isEditing) {
    if (!payload.headers?.length || !payload.rows?.length) {
      return (
        <div className="rounded-lg border border-dashed p-8 text-center text-muted-foreground">
          <p>Tabel kosong - klik untuk mengedit</p>
        </div>
      );
    }

    return (
      <div className="overflow-x-auto rounded-lg border">
        <table className="w-full">
          <thead className="bg-muted/50">
            <tr>
              {payload.headers.map((header, i) => (
                <th key={i} className="px-4 py-3 text-left text-sm font-semibold border-b">
                  {header || <span className="text-muted-foreground italic">-</span>}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {payload.rows.map((row, i) => (
              <tr key={i} className={cn(i % 2 === 0 ? 'bg-background' : 'bg-muted/20')}>
                {row.map((cell, j) => (
                  <td key={j} className="px-4 py-3 text-sm border-b last:border-b-0">
                    {cell || <span className="text-muted-foreground italic">-</span>}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  // Edit mode handlers
  const updateHeader = (index: number, value: string) => {
    setHeaders((prev) => prev.map((h, i) => (i === index ? value : h)));
  };

  const updateCell = (rowIndex: number, colIndex: number, value: string) => {
    setRows((prev) =>
      prev.map((row, ri) => (ri === rowIndex ? row.map((cell, ci) => (ci === colIndex ? value : cell)) : row))
    );
  };

  const addColumn = () => {
    const newColNum = headers.length + 1;
    setHeaders((prev) => [...prev, `Kolom ${newColNum}`]);
    setRows((prev) => prev.map((row) => [...row, '']));
  };

  const removeColumn = (index: number) => {
    if (headers.length <= 1) return;
    setHeaders((prev) => prev.filter((_, i) => i !== index));
    setRows((prev) => prev.map((row) => row.filter((_, i) => i !== index)));
  };

  const addRow = () => {
    setRows((prev) => [...prev, new Array(headers.length).fill('')]);
  };

  const removeRow = (index: number) => {
    if (rows.length <= 1) return;
    setRows((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSave = () => {
    onSave({ headers, rows });
  };

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex flex-wrap items-center gap-2 p-3 bg-muted/30 rounded-lg border">
        <span className="text-sm font-medium text-muted-foreground mr-2">Ukuran:</span>
        <div className="flex items-center gap-1 text-sm">
          <Columns className="h-4 w-4 text-muted-foreground" />
          <span>{headers.length} kolom</span>
        </div>
        <span className="text-muted-foreground">×</span>
        <div className="flex items-center gap-1 text-sm">
          <Rows className="h-4 w-4 text-muted-foreground" />
          <span>{rows.length} baris</span>
        </div>
        <div className="flex-1" />
        <Button type="button" variant="outline" size="sm" onClick={addColumn}>
          <Plus className="h-3 w-3 mr-1" />
          Kolom
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={addRow}>
          <Plus className="h-3 w-3 mr-1" />
          Baris
        </Button>
      </div>

      {/* Table Editor */}
      <div className="overflow-x-auto border rounded-lg">
        <table className="w-full min-w-[400px]">
          {/* Header Row */}
          <thead className="bg-muted/50">
            <tr>
              <th className="w-8 p-2 border-b border-r bg-muted/70">
                <span className="sr-only">Row actions</span>
              </th>
              {headers.map((header, i) => (
                <th key={i} className="p-0 border-b border-r last:border-r-0 min-w-[120px]">
                  <div className="flex items-center">
                    <Input
                      value={header}
                      onChange={(e) => updateHeader(i, e.target.value)}
                      placeholder={`Kolom ${i + 1}`}
                      className="border-0 rounded-none bg-transparent font-semibold text-sm h-10 focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:bg-primary/5"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 shrink-0 text-muted-foreground hover:text-destructive"
                      onClick={() => removeColumn(i)}
                      disabled={headers.length <= 1}
                      title="Hapus kolom"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </th>
              ))}
            </tr>
          </thead>

          {/* Data Rows */}
          <tbody>
            {rows.map((row, ri) => (
              <tr key={ri} className={cn(ri % 2 === 0 ? 'bg-background' : 'bg-muted/10')}>
                <td className="p-1 border-b border-r bg-muted/30 text-center">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-muted-foreground hover:text-destructive"
                    onClick={() => removeRow(ri)}
                    disabled={rows.length <= 1}
                    title="Hapus baris"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </td>
                {row.map((cell, ci) => (
                  <td key={ci} className="p-0 border-b border-r last:border-r-0">
                    <Input
                      value={cell}
                      onChange={(e) => updateCell(ri, ci, e.target.value)}
                      placeholder="..."
                      className="border-0 rounded-none bg-transparent text-sm h-10 focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:bg-primary/5"
                    />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Help text */}
      <p className="text-xs text-muted-foreground">
        💡 Tip: Klik pada sel untuk mengedit. Gunakan tombol + untuk menambah kolom/baris.
      </p>

      {/* Action Buttons */}
      <div className="flex gap-2 pt-2">
        <Button onClick={handleSave} disabled={isSaving}>
          {isSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Simpan
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Batal
        </Button>
      </div>
    </div>
  );
}
