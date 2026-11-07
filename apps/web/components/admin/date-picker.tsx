"use client";

import { useState } from "react";
import { CalendarIcon } from "lucide-react";
import { id as indonesianLocale } from "react-day-picker/locale";

import { formatDateLong, parseDateOnly, toDateOnly } from "@/components/admin/date";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";

type DatePickerProps = Omit<
  React.ComponentProps<typeof Button>,
  "value" | "onChange" | "children"
> & {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
};

export function DatePicker({
  value,
  onChange,
  placeholder = "Pilih tanggal",
  className,
  ...props
}: DatePickerProps) {
  const [open, setOpen] = useState(false);
  const selected = parseDateOnly(value);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          className={cn(
            "w-full justify-start font-normal",
            !selected && "text-muted-foreground",
            className,
          )}
          {...props}
        >
          <CalendarIcon aria-hidden="true" />
          {selected ? formatDateLong(value) : placeholder}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          locale={indonesianLocale}
          selected={selected}
          defaultMonth={selected}
          onSelect={(date) => {
            if (date) {
              onChange(toDateOnly(date));
              setOpen(false);
            }
          }}
          autoFocus
        />
      </PopoverContent>
    </Popover>
  );
}
