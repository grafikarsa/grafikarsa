"use client";

import { useState } from "react";
import { ChevronsUpDown } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import type { Major } from "@/lib/api/gen/models";

/**
 * Multi-select combobox for teacher majors, built from the shadcn Command and
 * Popover primitives.
 */
export function MajorCombobox({
  majors,
  value,
  onChange,
  disabled = false,
  placeholder = "Pilih jurusan",
}: {
  majors: Major[];
  value: string[];
  onChange: (value: string[]) => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  const [open, setOpen] = useState(false);
  const selected = majors.filter((major) => value.includes(major.id));

  const toggle = (majorId: string) => {
    if (value.includes(majorId)) {
      onChange(value.filter((id) => id !== majorId));
      return;
    }

    onChange([...value, majorId]);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className="w-full justify-between font-normal"
        >
          <span className="min-w-0 truncate text-left">
            {selected.length > 0
              ? selected.map((major) => major.shortName).join(", ")
              : placeholder}
          </span>
          <ChevronsUpDown aria-hidden="true" className="opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-(--radix-popover-trigger-width) p-0">
        <Command>
          <CommandInput placeholder="Cari jurusan…" />
          <CommandList>
            <CommandEmpty>Tidak ada jurusan.</CommandEmpty>
            <CommandGroup>
              {majors.map((major) => {
                const isSelected = value.includes(major.id);

                return (
                  <CommandItem
                    key={major.id}
                    value={`${major.name} ${major.code}`}
                    data-checked={isSelected}
                    onSelect={() => toggle(major.id)}
                  >
                    <span className="min-w-0 flex-1 truncate">
                      {major.name}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {major.shortName}
                    </span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

/** Renders the selected majors as compact badges. */
export function MajorBadges({
  majors,
  majorIds,
}: {
  majors: Major[];
  majorIds: string[];
}) {
  const selected = majors.filter((major) => majorIds.includes(major.id));

  if (selected.length === 0) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {selected.map((major) => (
        <Badge key={major.id} variant="secondary">
          {major.shortName}
        </Badge>
      ))}
    </div>
  );
}
