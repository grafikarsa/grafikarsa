"use client";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// SelectField: pembungkus tipis primitif shadcn Select untuk daftar opsi
// sederhana (nilai + label), agar halaman tidak mengulang komposisi Radix.
export function SelectField({
  value,
  onValueChange,
  options,
  placeholder = "Pilih...",
  className,
  disabled,
}: {
  value: string;
  onValueChange: (v: string) => void;
  options: { value: string; label: string }[];
  placeholder?: string;
  className?: string;
  disabled?: boolean;
}) {
  return (
    <Select value={value || undefined} onValueChange={(v) => onValueChange(v ?? "")} disabled={disabled}>
      <SelectTrigger className={className}>
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
