"use client";

import { type ComponentProps, useState } from "react";
import { Eye, EyeOff } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

/**
 * Password field with a show/hide toggle. It forwards every prop (including
 * the ones injected by shadcn's FormControl) to the underlying input.
 */
export function PasswordInput({
  className,
  ...props
}: ComponentProps<typeof Input>) {
  const [visible, setVisible] = useState(false);
  const Icon = visible ? EyeOff : Eye;

  return (
    <div className="relative">
      <Input
        type={visible ? "text" : "password"}
        className={cn("pr-9", className)}
        {...props}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="absolute top-1/2 right-0.5 -translate-y-1/2"
        aria-label={visible ? "Sembunyikan sandi" : "Tampilkan sandi"}
        onClick={() => setVisible((current) => !current)}
      >
        <Icon aria-hidden="true" />
      </Button>
    </div>
  );
}
